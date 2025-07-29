# deployer-service
__This is beta software in early development!__
A Go webhook listener and deployment automation tool. This utility is optimized for a simple tech stack of a single node AWS EC2 instance running Amazon Linux 2, and orchestrating a minimal number of containers via Docker Compose.  JWT signed Webhooks from GitHub actions trigger the deploy service running on the EC2 instance to pull new containers from AWS ECR and to relaunch then with Compose.

## 🚀 Release
To publish a new release of the deployer-service, push a new Git tag that follows semantic versioning.

🔧 To create a release, check the latest tag and assign the next sequential tag:
```bash
# Fetch all tags from remote
git fetch --tags

# Get the latest tag (assumes tags follow v<major>.<minor>.<patch> format)
latest_tag=$(git tag --sort=-v:refname | head -n 1)
echo "Latest tag: $latest_tag"

# Determine the next tag using semver best practices

# Tag your latest commit and push
git tag <next_tag>
git push origin <next_tag>
```

GitHub Actions will automatically:
1. Build a statically linked Linux binary (linux/amd64) for Amazon Linux 2
2. Embed the release version into the binary
3. Create a GitHub Release associated with the tag
4. Upload the compiled binary (deployer-service) as a release asset
    
NOTE - Builds using CGO_ENABLED=0 to ensure compatibility with Amazon Linux 2.

✅ Example
```bash
# Assuming the latest tag is v0.0.13, the next tag would be v0.0.14
git tag v0.0.14
git push origin v0.0.14
```

Visit the Releases page on GitHub to download the latest version or view past versions.

## MVP
All of the following tests assume the `export IP=<EC2_IP_ADDRESS>:8686` has been set.
- [x] Set up minimal health-check + american central time zone logging. Test with
```bash
curl http://$IP/health
```
- [x] Add the essential `/deploy` endpoint with base64 encoded parameter string equal to the service name and the repository uri for the container. Test with 
```bash
echo -n '{"service":"frontend","image":"123456789.dkr.ecr.us-west-2.amazonaws.com/frontend:latest"}' | base64
```
Then cut & paste output into
```bash
curl -X POST http://$IP/deploy -d 'eyJzZXJ2aWNlIjoiZnJvbnRlbmQiLCJpbWFnZSI6IjEyMzQ1Njc4OS5ka3IuZWNyLnVzLXdlc3QtMi5hbWF6b25hd3MuY29tL2Zyb250ZW5kOmxhdGVzdCJ9'
```
- [x] Build a GitHub action in a different project and test that the webhook hits the `/deploy` endpoint.
- [x] 🔒 Add JWT validation (via http://github.com/zacsketches/go-jwt for compatibility with GitHub actions and AWS Linux 2) Test by setting up a `/test-keys` folder and including `private.pem, public.pem` and `wrong-private.pem`. Then test various combinations of the deploy handler to ensure it validates for the correct keys and refuses for the wrong ones.
```bash
jwt sign --key ./test-keys/private.pem > good-token.jwt
jwt sign --key ./test-keys/wrong-private.pem > bad-token.jwt

curl -X POST http://<ec2-ip>:8686/deploy \
  -H "Authorization: Bearer $(cat <good|bad>-token.jwt)" \
  -H "Content-Type: text/plain" \
  --data "$(echo -n '{"service":"frontend","image":"$IMAGE"}' | base64)"

```
- [x] Add Docker Compose control logic in the `exec.go` file to login to ECR, pull images, and then run `docker compose -f <path/to/compose> up -d` to implement full workflow.

--------------

- [ ] Deploy the `deployer` on a test project and use it to implement CD for a while.  Then come back and make this better after learning a few thing.