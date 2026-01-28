FROM golang:1.22-bookworm

RUN apt-get update && apt-get install -y gnupg && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 vault
ENV GNUPGHOME=/home/vault/.gnupg

RUN mkdir -p /home/vault/.gnupg && \
    chown -R vault:vault /home/vault && \
    chmod 700 /home/vault/.gnupg

RUN echo "allow-loopback-pinentry" > /home/vault/.gnupg/gpg-agent.conf && \
    echo "pinentry-mode loopback" > /home/vault/.gnupg/gpg.conf && \
    chown vault:vault /home/vault/.gnupg/gpg-agent.conf /home/vault/.gnupg/gpg.conf

WORKDIR /app
COPY . .

RUN go build -o vaultseal && \
    chown -R vault:vault /app && \
    chmod 600 *.asc

USER vault
EXPOSE 8080
CMD ["./vaultseal"]