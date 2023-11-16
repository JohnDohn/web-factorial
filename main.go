// main
package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type param struct {
	A int `json:"a"`
	B int `json:"b"`
}

type ParamChecker struct {
	handler http.Handler
}

var factorialChanIn = make(chan int)
var factorialChanOut = make(chan int)
var baseURL = "/calculate"

func factorial(cin chan int, cout chan int) {
	for {
		f := 1
		v := <-cin

		for i := 1; i <= v; i++ {
			f = f * i
		}

		cout <- f
	}
}

func calculateHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var pin, pout param

	err := json.NewDecoder(r.Body).Decode(&pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Print("bad request")
		return
	}

	log.Printf(`<= {"a": %v, "b": %v}`, pin.A, pin.B)

	factorialChanIn <- pin.A
	pout.A = <-factorialChanOut

	factorialChanIn <- pin.B
	pout.B = <-factorialChanOut

	log.Printf(`=> {"a": %v, "b": %v}`, pout.A, pout.B)

	err = json.NewEncoder(w).Encode(pout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (pc *ParamChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.RequestURI == baseURL {
		pin := param{A: -1, B: -1}
		body, _ := ioutil.ReadAll(r.Body)

		err := json.NewDecoder(bytes.NewReader(body)).Decode(&pin)
		if err != nil || pin.A < 0 || pin.B < 0 {
			http.Error(w, `{ "error":"Incorrect input"}`, http.StatusBadRequest)
			log.Print("paramChecker: skipping request")
			return
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	log.Print("paramChecker: forwarding request to the next handler")

	pc.handler.ServeHTTP(w, r)
}

func NewParamChecker(handlerToWrap http.Handler) *ParamChecker {
	return &ParamChecker{handlerToWrap}
}

func main() {
	addr := "localhost:8989"

	log.SetPrefix("web-factorial: ")
	log.SetFlags(0)

	go factorial(factorialChanIn, factorialChanOut)

	router := httprouter.New()
	router.POST(baseURL, calculateHandler)

	log.Printf("Serving http requests at %s", addr)

	log.Fatal(http.ListenAndServe(addr, NewParamChecker(router)))
}
