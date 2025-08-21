func llamarAPI1(w http.ResponseWriter, r *http.Request) {
    resp, err := http.Get("http://192.168.122.168:8080/api1/202204496")
    if err != nil {
        http.Error(w, "Error llamando a API1", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    var api1Msg Respuesta
    json.Unmarshal(body, &api1Msg)
    respAPI3 := Respuesta{
        Mensaje: "Hola, responde la API: API3 en la VM2, desarrollada por el estudiante Rodrigo Castro con carnet: 202204496. Llamé a API1 y respondió: " + api1Msg.Mensaje,
    }
    json.NewEncoder(w).Encode(respAPI3)
}

func llamarAPI2(w http.ResponseWriter, r *http.Request) {
    resp, err := http.Get("http://192.168.122.168:8081/api2/202204496") // Cambia 8080 a 8081
    if err != nil {
        http.Error(w, "Error llamando a API2", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    var api2Msg Respuesta
    json.Unmarshal(body, &api2Msg)
    respAPI3 := Respuesta{
        Mensaje: "Hola, responde la API: API3 en la VM2, desarrollada por el estudiante Rodrigo Castro con carnet: 202204496. Llamé a API2 y respondió: " + api2Msg.Mensaje,
    }
    json.NewEncoder(w).Encode(respAPI3)
}