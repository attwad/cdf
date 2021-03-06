FROM golang:1.9

WORKDIR /go/src/app
COPY . .

RUN go-wrapper download
RUN go-wrapper install

# Install sox.
RUN apt-get clean && apt-get -y update && apt-get install -y sox libsox-fmt-mp3

# Provide a sensible default run command.
CMD ["go-wrapper", "run", "--project_id=college-de-france", "--bucket=healthy-cycle-9484", "--sox_path=sox", "--elastic_address=http://127.0.0.1:9200"]
