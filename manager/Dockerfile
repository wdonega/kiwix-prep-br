# Imagem única: kiwix-serve oficial + gerenciador.
#
#   docker build -t prep-wiki-br manager/
#   docker buildx build --platform linux/amd64,linux/arm64 -t prep-wiki-br manager/
#
# O gerenciador é compilado na máquina que faz o build, já para a arquitetura de destino
# (o Go faz isso sozinho), então gerar a imagem ARM num PC não precisa de emulação.

FROM --platform=$BUILDPLATFORM golang:1-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY *.go catalogo.json ./
COPY web ./web
RUN CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./... && \
    CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /prep-manager .

FROM ghcr.io/kiwix/kiwix-serve:3.8.2@sha256:e478554a9920412eeb5ce1d73f24e4b28445ebed233187cdd03c623c32be46a5
COPY --from=build /prep-manager /usr/local/bin/prep-manager
ENV PREP_DADOS=/data \
    PREP_ENDERECO=0.0.0.0:8080 \
    PREP_KIWIX_PORTA=8081
EXPOSE 8080
VOLUME /data
# Sobe como root só para preparar a pasta de dados; o gerenciador troca para um usuário comum logo
# na partida, antes de abrir a porta ou iniciar o kiwix-serve (veja privilegio_linux.go).
# O gerenciador é o processo principal do container (sem o dumb-init da imagem do Kiwix, que ficaria
# rodando como root): ele mesmo trata os sinais de parada e acompanha o kiwix-serve.
USER 0
ENTRYPOINT ["/usr/local/bin/prep-manager"]
