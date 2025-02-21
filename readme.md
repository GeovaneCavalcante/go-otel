# Como rodar a aplicação

Siga os passos abaixo para executar a aplicação utilizando os comandos do [Makefile](https://github.com/GeovaneCavalcante/go-otel/blob/main/Makefile):

## Iniciando os serviços

1. **Iniciar dependências**
   ```
   make up-depedences
   ```
   Este comando inicia os containers definidos no docker-compose.yml.

2. **Executar API de pagamento**
   ```
   make run-payment-api
   ```
   Executa a API de pagamento com o comando `go run payment/main.go`.

3. **Executar API de autenticação**
   ```
   make run-auth-api
   ```
   Executa a API de autenticação com o comando `go run authorization/main.go`.

4. **Executar serviço faker**
   ```
   make run-faker
   ```
   Executa o serviço de dados simulados com o comando `go run faker/main.go`.

## Acessar métricas

Para visualizar o dashboard com as métricas:

1. Acesse [localhost:3000](http://localhost:3000)
2. Faça login no Grafana usando:
   - Usuário: `admin`
   - Senha: `admin`
3. Navegue para `Dashboards` → `Import`
4. Copie o conteúdo do arquivo [grafana.json](https://github.com/GeovaneCavalcante/go-otel/blob/main/docs/dashes/grafana.json) e cole no campo de texto
5. Clique em `Load` para visualizar o dashboard

## Acessar traces

Para visualizar as traces:

1. Acesse [localhost:9411](http://localhost:9411)
2. Clique em `Run Query` para visualizar as traces disponíveis
