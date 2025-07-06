# deployer-service
A Go webhook listener and deployment automation tool. This utility is optimized for a simple tech stack of a single node AWS EC2 instance running Amazon Linux 2, and orchestrating a minimal number of containers via Docker Compose.  JWT signed Webhooks from GitHub actions trigger the deploy service to pull new containers from AWS ECR and to relaunch then with Compose.

## MVP
All of the following tests assume the `export IP=<EC2_IP_ADDRESS>:8686` has been set.
- [x] Set up minimal health-check + american central time zone logging. Test with
```bash
curl http://$IP/health
```
- [x] Add the central `/deploy` endpoint with base64 encoded parameter string equal to the service name and the repository uri for the container. Test with 
```bash
echo -n '{"service":"frontend","image":"123456789.dkr.ecr.us-west-2.amazonaws.com/frontend:latest"}' | base64
```
Then cut & paste output into
```bash
curl -X POST http://$IP/deploy -d 'eyJzZXJ2aWNlIjoiZnJvbnRlbmQiLCJpbWFnZSI6IjEyMzQ1Njc4OS5ka3IuZWNyLnVzLXdlc3QtMi5hbWF6b25hd3MuY29tL2Zyb250ZW5kOmxhdGVzdCJ9'
```
- [x] Build a GitHub action in a different project and test that the webhook hits the `/deploy` endpoint.
- [ ] 🔒 Add JWT validation (via http://github.com/zacsketches/go-jwt for compatibility with GitHub action and AWS Linux 2) Test by setting up a `/test-keys` folder and including `private.pem, public.pem` and `wrong-private.pem`. Then test various combinations of the deploy handler to ensure it validates for the correct keys and refuses for the wrong ones.
```bash
jwt sign --key ./test-keys/private.pem > good-token.jwt
jwt sign --key ./test-keys/wrong-private.pem > bad-token.jwt

curl -X POST http://<your-ec2-ip>:8686/deploy \
  -H "Authorization: Bearer $(cat [good|bad]token.jwt)" \
  -H "Content-Type: text/plain" \
  --data "$(echo -n '{"service":"frontend","image":"foo.aws.ecr.bar"}' | base64)"

```
- [ ] 🐳 Add Docker Compose control logic
- [ ] 🚀 Create public repo or internal repo (e.g., fathom5/deployer-service)
- [ ] 🧪 Set up GitHub Actions to build binary and push to S3

## Helpful Command Line Foo
