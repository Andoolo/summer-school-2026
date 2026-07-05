// Веб dev-сервер на 8081, чтобы не конфликтовать с backend API на :8080.
// Приложение из браузера ходит на API по http://localhost:8080.
config.devServer = config.devServer || {};
config.devServer.port = 8081;
config.devServer.open = false;
