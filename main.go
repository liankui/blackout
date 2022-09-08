package main

import (
    "encoding/json"
    "io/ioutil"
    "liankui/blackout/sliding_window"
    "log"
    "net/http"
    "strconv"
    "time"
)

var (
    X         = 2
    R         = 500
    NeedTotal = 1000
    sw        *sliding_window.SlidingWindow
)

func main() {
    // Create a SlidingWindow that has a window of 1 hour, with a granulity of 1 minute.
    sw = sliding_window.MustNew(time.Hour, time.Minute)
    defer sw.Stop()

    errc := make(chan error)
    go func() {
        mux := http.NewServeMux()
        mux.HandleFunc("/age", multiplexer)
        mux.HandleFunc("/car", multiplexer)
        mux.HandleFunc("/rate", multiplexer)
        mux.HandleFunc("/buffer", multiplexer)

        errc <- http.ListenAndServe(":8081", mux)
    }()

    time.Sleep(100 * time.Millisecond)

    done := make(chan bool)
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    go func() {
        for {
            select {
            case <-done:
                return
            case <-ticker.C:
                // check inventory size
                buffer, err := GetBuffer("http://127.0.0.1:8081/buffer")
                //fmt.Printf("buffer=%v\n", buffer)
                if err != nil {
                    errc <- err
                }
                for buffer < NeedTotal {
                    // todo creates a single vehicle
                    buffer++
                }
            }
        }
    }()

    log.Print("server", <-errc)
    done <- true
}

func multiplexer(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        getHandler(w, r)
    case http.MethodPost:
        postHandler(w, r)
    }
}

func getHandler(w http.ResponseWriter, r *http.Request) {
    respMap := make(map[string]interface{})
    switch r.URL.Path {
    case "/age":
        respMap["age"] = X
    case "/car":
        sw.Add(1)
        total, _ := sw.Total(time.Minute)
        R += int(total)
        respMap["car"] = strconv.FormatInt(time.Now().UnixNano(), 10)
    case "/rate":
        respMap["rate"] = R
    case "/buffer":
        respMap["buffer"] = R * X
    }
    resp, _ := json.Marshal(respMap)
    w.Write(resp)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := ioutil.ReadAll(r.Body)
    switch r.URL.Path {
    case "/age":
        var m map[string]int
        _ = json.Unmarshal(body, &m)
        X = m["age"]
    }
    w.Write([]byte(`{"ok":true}`))
}

func GetBuffer(url string) (int, error) {
    resp, err := http.Get(url)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return 0, err
    }
    var m map[string]int
    _ = json.Unmarshal(body, &m)
    return m["buffer"], nil
}
