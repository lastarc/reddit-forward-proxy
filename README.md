# reddit-forward-proxy

Simple request proxy with Zitadel authentication intended for use in 
[glanceapp/glance](https://github.com/glanceapp/glance)

## Why

Reddit blocks IP of VPS where the glance app is hosted. With this, 
the request can be proxied through self-hosted infrastructure while keeping it private.

## How

1. Setup [Zitadel](https://zitadel.com)

   1. Register and create a project if you don't have one 
   
   2. Open Roles, create new one (ex. reddit-forward-proxy-access)
   
   3. Go to General, and create an application 
      1. Give it a name (ex. reddit-forward-proxy)
      2. Select API
      3. Keep 'Private Key JWT'
      4. Click create (no need to copy clientId)
      5. Add new Key (JSON, you can leave expiration empty) and download it

   4. Go to Users tab > Service Users
      1. Create new one
      2. Fill necessary fields (ex. glance-app)
      3. Keep Access Token Type 'Bearer'
      4. Afterwards, open Personal Access Tokens, generate new one, 
      copy and save the token
      5. Authorizations > New > select your project > add previously created
      (1.2) role, and save it

2. Prepare `reddit-forward-proxy`

   1. (Run the followings in machine with the "safe" IP)

   2. Clone the repo and `cd` into it
   
   ```shell
   git clone https://github.com/lastarc/reddit-forward-proxy.git
   cd reddit-forward-proxy
   ```

   3. Build the docker image

   ```shell
   docker build . -t reddit-forward-proxy
   ```
   
   4. Copy/move the key file (1.3.5) to this machine
   
   5. Run the image
   
   ```shell
   docker run -it -rm \
      -v /path/to/key/xxxxxxxxxxxxxxxxxx.json:/app/key.json
      -p 8089:8089
      reddit-forward-proxy
      /app/server --domain yourdomain.zitadel.cloud --key /app/key.json
   ```

   6. (Optional) Setup a reverse proxy (ex. cloudflared)

3. Add `request-url-template: <reddit-forward-proxy access url>/api/proxy?apiKey=<PAT from 1.4.4>&url={REQUEST-URL}` 
to your `glance.yml`

```diff
...
           - type: reddit
             subreddit: selfhosted
+            request-url-template: reddit-forward-proxy.mydomain.com/api/proxy?apiKey=COp...jYI&url={REQUEST-URL}
```

4. Done?

## To do (possibly)

- [ ] Make auth optional
- [ ] Fix up defaults for docker image 
(`... /app/server --domain yourdomain.zitadel.cloud --key /app/key.json` is too verbose)
