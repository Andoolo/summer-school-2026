// Картинг-версия: веб dev-сервер на 8091 (API картинга на :8090), чтобы не конфликтовать с «Волной».
config.devServer = config.devServer || {};
config.devServer.port = 8091;
config.devServer.open = false;
