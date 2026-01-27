FROM golang:1.22-bookworm

RUN apt-get update && apt-get install -y gnupg && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 vault
ENV GNUPGHOME=/home/vault/.gnupg

# สร้างโฟลเดอร์และตั้งสิทธิ์ให้ถูกต้องก่อนรัน
RUN mkdir -p /home/vault/.gnupg && \
    chown -R vault:vault /home/vault && \
    chmod 700 /home/vault/.gnupg

WORKDIR /app
COPY . .
RUN go build -o vaultseal

USER vault

EXPOSE 8080
CMD ["./vaultseal"]