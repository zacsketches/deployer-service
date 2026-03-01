# deployer-service

`deployer-service` is a small Go HTTP daemon that runs on an EC2 host and performs Docker Compose deployments when it receives a signed webhook.

## What it does

- Exposes HTTP endpoints:
  - `GET /health`
  - `GET /version`
  - `GET /config`
  - `POST /deploy` (JWT required)
  - `POST /logout` (debug mode only)
- On startup it attempts:
  1. AWS ECR login
  2. `docker compose up -d`
- On deploy request it runs:
  1. `docker compose pull <service>`
  2. If pull fails, ECR login + retry pull once
  3. `docker compose up -d`

## Runtime requirements

The host must have:

- Docker Engine + Compose plugin (`docker compose` command)
- AWS CLI v2 (`aws` command)
- Network access to AWS ECR
- IAM permissions that allow ECR auth/pull (for the instance role or credentials used by the service)
- A Docker Compose file already present on disk (for example in `/opt/deployer-service/compose/docker-compose.yml`)
- An RSA public key file for JWT verification

## Required environment variables

The service **fails fast at startup** if any of these are missing or invalid:

- `JWT_PUBLIC_KEY_PATH` – path to RSA public key PEM used to verify `Authorization: Bearer <jwt>`
- `DOCKER_COMPOSE_FILE` – full path to your compose file
- `AWS_REGION` – AWS region for ECR login
- `ECR_DOMAIN` – ECR registry domain, e.g. `<account-id>.dkr.ecr.us-west-2.amazonaws.com`
- `DEBUG_MODE` – must be exactly `true` or `false`

Optional:

- `PORT` – defaults to `8686`

## Expected host layout (example)

You can choose different paths, but this layout matches the sample systemd unit:

```text
/opt/deployer-service/
  deployer                        # binary
  deployer.pub                    # JWT RSA public key
  compose/
    docker-compose.yml            # compose file used by DOCKER_COMPOSE_FILE
```

## Build

Local build:

```bash
go build -o deployer .
```

Amazon Linux compatible static build:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-X=main.version=$(git describe --tags --always --dirty)" \
  -o deployer .
```

## systemd setup (EC2)

1. Copy binary and config files to the EC2 host (example paths above).
2. Install a unit file at `/etc/systemd/system/deployer.service`.
3. Reload systemd, enable, and start the service.

Example unit (adapt values for your environment):

```ini
[Unit]
Description=Deployer Service
After=docker.service
Requires=docker.service

[Service]
WorkingDirectory=/opt/deployer-service
ExecStart=/opt/deployer-service/deployer
Environment="JWT_PUBLIC_KEY_PATH=/opt/deployer-service/deployer.pub"
Environment="DOCKER_COMPOSE_FILE=/opt/deployer-service/compose/docker-compose.yml"
Environment="AWS_REGION=us-west-2"
Environment="ECR_DOMAIN=<account-id>.dkr.ecr.us-west-2.amazonaws.com"
Environment="DEBUG_MODE=false"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable/start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable deployer
sudo systemctl restart deployer
sudo systemctl status deployer --no-pager
```

Follow logs:

```bash
sudo journalctl -u deployer -f
```

## EC2 `user_data.sh` example

The snippet below is intended for an Amazon Linux 2 EC2 instance. It installs runtime dependencies, pulls the latest `deployer-service` release artifact from GitHub Releases, fetches `ECR_DOMAIN` from AWS SSM Parameter Store, writes a systemd unit, and starts the service.

> Replace placeholder values before use: `AWS_REGION`, `ECR_DOMAIN_SSM_PARAM`, `GITHUB_REPO`, and the JWT public key content.

```bash
#!/bin/bash
set -euo pipefail

# ---- Required configuration ----
AWS_REGION="<REPLACE ME>"
ECR_DOMAIN_SSM_PARAM="<REPLACE ME>"
GITHUB_REPO="zacsketches/deployer-service"           
DEPS_DIR="/opt/deployer-service"
COMPOSE_FILE="$DEPS_DIR/compose/docker-compose.yml"
PUBLIC_KEY_PATH="$DEPS_DIR/deployer.pub"
DEBUG_MODE="false"

# ---- OS packages ----
yum update -y
yum install -y docker awscli curl
systemctl enable docker
systemctl start docker

# ---- Service directory ----
mkdir -p "$DEPS_DIR/compose"

# ---- Resolve ECR domain from SSM Parameter Store ----
# Instance role needs: ssm:GetParameter on $ECR_DOMAIN_SSM_PARAM
ECR_DOMAIN="$(aws ssm get-parameter \
  --name "$ECR_DOMAIN_SSM_PARAM" \
  --with-decryption \
  --region "$AWS_REGION" \
  --query 'Parameter.Value' \
  --output text)"

# ---- Deployer binary (latest release asset named 'deployer-service') ----
curl -fL "https://github.com/${GITHUB_REPO}/releases/latest/download/deployer-service" -o "$DEPS_DIR/deployer"
chmod +x "$DEPS_DIR/deployer"

# ---- JWT public key ----
cat > "$PUBLIC_KEY_PATH" <<'EOF'
-----BEGIN PUBLIC KEY-----
REPLACE_WITH_YOUR_RSA_PUBLIC_KEY
-----END PUBLIC KEY-----
EOF
chmod 0644 "$PUBLIC_KEY_PATH"

# ---- Your compose file ----
# Place your docker-compose.yml at $COMPOSE_FILE.
# You can write it here in user_data, copy it from S3, or provision it with another step.

# ---- systemd unit ----
cat > /etc/systemd/system/deployer.service <<EOF
[Unit]
Description=Deployer Service
After=docker.service
Requires=docker.service

[Service]
WorkingDirectory=$DEPS_DIR
ExecStart=$DEPS_DIR/deployer
Environment="JWT_PUBLIC_KEY_PATH=$PUBLIC_KEY_PATH"
Environment="DOCKER_COMPOSE_FILE=$COMPOSE_FILE"
Environment="AWS_REGION=$AWS_REGION"
Environment="ECR_DOMAIN=$ECR_DOMAIN"
Environment="DEBUG_MODE=$DEBUG_MODE"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable deployer
systemctl restart deployer
systemctl status deployer --no-pager || true
```

The release artifact is named `deployer-service`; this script downloads it and saves it as `$DEPS_DIR/deployer` to match `ExecStart`.

## Webhook contract

### Authentication

- `/deploy` requires `Authorization: Bearer <jwt>`
- JWT must verify against `JWT_PUBLIC_KEY_PATH`
- The issuer (`iss`) claim is logged

### Request body format

Body must be `text/plain` containing **base64-encoded JSON**:

```json
{"service":"frontend"}
```

Current behavior notes:

- `service` is used for `docker compose pull <service>`
- Image names/tags come from the compose file referenced by `DOCKER_COMPOSE_FILE`

Example request:

```bash
payload=$(echo -n '{"service":"frontend"}' | base64)

curl -X POST http://<ec2-ip>:8686/deploy \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: text/plain" \
  --data "$payload"
```

## Health, version, and config checks

```bash
curl http://<ec2-ip>:8686/health
curl http://<ec2-ip>:8686/version
curl http://<ec2-ip>:8686/config
```

## Debug logout endpoint

When `DEBUG_MODE=true`, `POST /logout` runs `docker logout $ECR_DOMAIN`.
This is useful for testing the pull-fails-then-relogin path.

```bash
curl -X POST http://localhost:8686/logout
```

In production, set `DEBUG_MODE=false` to disable this behavior.

## Release notes

To create a release, push a semantic version tag (for example `v0.0.14`).
Utility script for latest tag:

```bash
bash bin/latest.sh
```

Typical flow:

```bash
git commit -a -m "Reason for latest change"
git push
git fetch --tags
git tag vX.Y.Z
git push origin vX.Y.Z
```

## Known limitations (current code behavior)

- Startup `runLogin()` / `runComposeUp()` errors are logged but do not stop daemon startup.
- JWT validation checks signature and presence of `iss`; it does not enforce custom claim rules beyond standard parsing.

---

If you use this behind a load balancer or reverse proxy, allow inbound traffic only from trusted webhook sources and keep the service port restricted by security group rules.