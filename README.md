# deployer-service
A Go webhook listener and deployment automation tool. This utility is optimized for a simple tech stack of a single node AWS EC2 instance running Amazon Linux 2, and orchestrating a minimal number of containers via Docker Compose.  JWT signed Webhooks from GitHub actions trigger the deploy service to pull new containers from AWS ECR and to relaunch then with Compose.

## MVP
- [x] Set up minimal health-check + american central time zone logging.
- [ ] Add the central `/deploy` endpoint with base64 encoded parameter string equal to the service name and the repository uri for the container. 
- [ ] 🔒 Add JWT validation (via http://github.com/zacsketches/go-jwt for compatibility with GitHub action and AWS Linux 2)
- [ ] 🐳 Add Docker Compose control logic
- [ ] 🚀 Create public repo or internal repo (e.g., fathom5/deployer-service)
- [ ] 🧪 Set up GitHub Actions to build binary and push to S3
