use actix_web::{post, web, App, HttpResponse, HttpServer, Responder};
use serde::{Deserialize, Serialize};
use reqwest::Client;

#[derive(Deserialize, Serialize)]
struct WeatherTweet {
    municipality: String, // Usamos String para validar enums manualmente
    temperature: i32,
    humidity: i32,
    weather: String,
}

#[post("/tweet")]
async fn post_tweet(tweet: web::Json<WeatherTweet>, client: web::Data<Client>) -> impl Responder {
    // Validar enums de municipality
    let valid_municipalities = vec!["mixco", "guatemala", "amatitlan", "chinautla"];
    if !valid_municipalities.contains(&tweet.municipality.as_str()) {
        return HttpResponse::BadRequest().body("Invalid municipality");
    }

    // Validar enums de weather
    let valid_weathers = vec!["sunny", "cloudy", "rainy", "foggy"];
    if !valid_weathers.contains(&tweet.weather.as_str()) {
        return HttpResponse::BadRequest().body("Invalid weather");
    }

    // Enviar al servicio Go (Deployment 1) vía HTTP
    let go_service_url = "http://go-service.dev.svc.cluster.local:8080/tweet";
    match client
        .post(go_service_url)
        .json(&tweet)
        .send()
        .await
    {
        Ok(resp) => {
            if resp.status().is_success() {
                HttpResponse::Ok().json(serde_json::json!({"status": "success"}))
            } else {
                HttpResponse::InternalServerError().body("Failed to forward to Go service")
            }
        }
        Err(_) => HttpResponse::InternalServerError().body("Error contacting Go service")
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let client = web::Data::new(Client::new());
    HttpServer::new(move || {
        App::new()
            .app_data(client.clone())
            .service(post_tweet)
    })
    .bind("0.0.0.0:8080")?
    .run()
    .await
}