# deployer-service
__This is beta software in early development!__

The `deployer-service` is Go webhook listener and deployment automation tool. This service is designed to support continuous delivery (CD) for a simple tech stack of a single node AWS EC2 instance running Amazon Linux 2, and orchestrating a minimal number of containers via Docker Compose.  JWT signed Webhooks from GitHub actions trigger the deploy service running on the EC2 instance to pull new containers from AWS ECR and to relaunch then with Compose.

This avoids complexity with full Kubernetes container orchestration and accelerates development on small projects that do not require fully K8S.

## Usage
The deployer service is designed to be run as a systemd service on an Amazon Elastic Cloud Compute (EC2) virtual machine. An example systemd service file is provided in the `/systmed` folder of the project. This service file is normally added to the `user_data.sh` script that configures the EC2 instance on standup. 

Note that there are several environment variables that must be present for the system to launch. Edit the provided service file to provide an AWS region and an AWS Elastic Container Registry (ECR) account id. The environment variable also include a __DEBUG_MODE variable that enables a `/logout` endpoint for testing__. This variable should be changed to `false` when shifting from development to production.

## Managing the Service
Since the `deployer-service` is the root of the CD pipeline, it has to be updated manually. There is a utility provided in the `/bin` folder of the project that should be loaded onto the EC2 instance. Running this utility from the instance upgrades the service to the newest version at the sysadmin's discretion. 

## Testing
The service uses docker to log into AWS ECR on launch, but the login credential from ECR time out after several hours. The service is resilient to this change and attempts to log back in on a failed `docker compose pull`. However, this is a difficult resiliency feature to test. So, the service enables a `/logout` endpoint when the `DEBUG_MODE=true` environment variable is set. With the `/logout` endpoint enabled you can curl the logout, and then send another deployment trigger to observe the retry logic. The following terminal windows are assumed to both be logged into an EC2 instance.
```
in terminal window 1 follow the service logs

journalctl -u deployer-service -f
```
```
in terminal window 2 log the service out of docker

curl -X POST http://localhost:8686/logout
```
Then make a small change to one of the system containers and push the update to CI/CD.  You should see the logs fail the first pull attempt and then log back in before trying the pull again.

## 🚀 Release
To publish a new release of the deployer-service, push a new Git tag that follows semantic versioning.

> **Before Creating a Release:**  
> Ensure that all features are pushed to remote and that the code is ready for production. The release process will automatically build the binary, embed the version, and create a GitHub Release.

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
    
NOTE 1 - Builds using CGO_ENABLED=0 to ensure compatibility with Amazon Linux 2.
NOTE 2 - There is a small utility script `bin/latest.sh` that can be used to determine the latest tag in the repository. It can be run with `bash bin/latest.sh`.

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