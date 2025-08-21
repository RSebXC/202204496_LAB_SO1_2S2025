package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type Respuesta struct {
    Mensaje string `json:"mensaje"`
}

func llamarAPI1(w http.ResponseWriter, r *http.Request) {
    resp, err := http.Get("http://localhost:8080/api1/202204496")
    if err != nil {
        http.Error(w, "Error llamando a API1", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    var api2Msg Respuesta
    json.Unmarshal(body, &api2Msg)
    respAPI1 := Respuesta{
        Mensaje: "Hola, responde la API: API2 en la VM1, desarrollada por el estudiante Rodrigo Castro con carnet: 202204496. Llamé a API2 y respondió: " + api1Msg.Mensaje,
    }
    json.NewEncoder(w).Encode(respAPI1)
}

func llamarAPI3(w http.ResponseWriter, r *http.Request) {
    resp := Respuesta{
        Mensaje: "Hola, responde la API: API3 en la VM3, desarrollada por el estudiante RodrigoC astro con carnet: 202204496",
    }
    json.NewEncoder(w).Encode(resp)
}

func main() {
    http.HandleFunc("/api2/202204496/llamar-api1", llamarAPI1)
    http.HandleFunc("/api2/202204496/llamar-api3", llamarAPI3)

    fmt.Println("API2 corriendo en puerto 8081...")
    http.ListenAndServe(":8081", nil)
}
