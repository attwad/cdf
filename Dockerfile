FROM golang:1.8

WORKDIR /go/src/app
COPY . .

RUN go-wrapper download
RUN go-wrapper install

# Install sox.
RUN apt-get clean && apt-get -y update && apt-get install -y sox libsox-fmt-mp3

# Provide a sensible default run command.
CMD ["go-wrapper", "run", "--project_id=college-de-france", "--bucket=healthy-cycle-9484", "--elastic_host=http://localhost:9200", "--sox_path=sox"]
