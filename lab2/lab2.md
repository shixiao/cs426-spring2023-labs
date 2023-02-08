# [Initial version] Lab 2: Deployment and operations

## Overview
In this lab, you will deploy the VideoRecService you built in Lab 1 using Kubernetes on DigitalOcean instances.
You will learn the basics of creating Docker images, Kubernetes deployments and services, and optimize
your VideoRecService implementation with connection pooling.

This lab will be operations-heavy and relatively light on coding.

**Note**: You will be using shared resources for this lab (a Docker registry and a Kubernetes cluster).
Please be conservative and careful with your usage to avoid overloading the shared resources for others. If you have any issues
with the cluster or shared tooling please reach out to us via Ed or email.

## Logistics
**Policies**
- This lab is meant to be an **individual** assignment. Please see the [Collaboration Policy](#collaboration-policy) for details.
- We will help you strategize how to debug but WE WILL NOT DEBUG YOUR CODE FOR YOU.
- Please keep and submit a time log of time spent and major challenges you've encountered. This may be familiar to you if you've taken CS323. See [Time logging](#time-logging) for details.

- Questions? post to Ed or email the teaching staff.
  - Mahdi Soleimani (mahdi.soleimani@yale.edu)
  - Ross Johnson (ross.johnson@yale.edu)
  - Xiao Shi (xiao.shi@aya.yale.edu)
  - Richard Yang (yry@cs.yale.edu)

**Submission deadline: 23:59 ET Wednesday February 15, 2023**

**Submission logistics**
Submit a `.tar.gz` archive named after your NetID via
Canvas. The Canvas assignment will be up a day or two before the deadline.

Your submission for this lab should include the following files:
 - Updated `server.go` and `server_lib.go` from video_rec_service/ from lab 1, after you make the connection pooling changes in Part C.
 - `Dockerfile` you completed in Part A.
 - `deployment.yml`, `service.yml` Kubernetes specs from Part B
 - `request.yml` CRD spec from Part B.
 - `result.yml`, a copy of the above `LoadgenResult` that you obtain from the cluster.
 - `time.log`, as in previous labs.
 - `discussions.md` -- a [Markdown](https://www.markdownguide.org/) file. The content can be plain text if you don't want to get fancy.

In addition, as per Part B instructions,
 - Leave a single `LoadgenResult` in your Kubernetes namespace. Delete any others. This run has to have at least one successful request (not 100% errors) to receive credit.
As per Part C instructions,
 - Leave your `VideoRecService` deployment running.

## Setup

You will need quite a few things installed for this lab. If you are in need of a unix environment, we have provided a base image that you can create a free VM instance on Google Cloud Platform (GCP), see [this section](#set-up-a-gcp-dev-environment) for instructions.

Our suggestion would be one the following:
- Use your personal device for everything (x86 MacOS or Linux preferred, Apple sillicon like M1 chips and WSL2 should also work);
- Use the Zoo for the Go coding part and use your personal device or the GCP dev env for the operational part (docker and kubectl);
- Use the GCP dev env for everything.

 1. Docker installed locally, with the ability to build images.
     - Follow the instructions here: https://docs.docker.com/get-docker/
 2. `kubectl`, the Kubernetes CLI, installed locally (version 1.20 or newer).
     - Follow instructions here: https://kubernetes.io/docs/tasks/tools/
     - **Tip**: kubernetes includes autocompletion for bash, zsh, etc.: https://kubernetes.io/docs/tasks/tools/included/
 3. Your Kubernetes config file with credentials which was shared to your Yale google drive (like your reading response document). You should see a file named `<NET-ID>.kubecfg`. This file contains your private key so don't share it with others!
     - You can place this file at `~/.kube/config` to use it by default, otherwise you can specify `kubectl --kubeconfig=<PATH-TO-FILE>` for all future steps. For other options on manging kubeconfig files, see the official documentation: https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/
     - If you don't see the config file, contact Xiao Shi (xiao.shi@aya.yale.edu)

Make sure these tools work before continuing to future steps. To test your Docker install, you can run
a test image such as `docker run --rm ubuntu uname` to test
the result of the `uname` command on a default `ubuntu` image.

To test your `kubectl` install, you should be able to run a simple list command like
```
kubectl get secret
```

Note that if you did not place your Kubernetes config into the default location `~/.kube/config`
you may need to run something like
```
KUBECONFIG=configs/sp2432.kubecfg kubectl get secret
```
or
```
kubectl --kubeconfig=configs/sp2432.kubecfg kubectl secret
```

Or set an environment variable for your current bash session (need to do this every time you open a new shell):
```
export KUBECONFIG=configs/sp2432.kubecfg

kubectl get secret
```

This should return something like:

```
NAME                  TYPE                                  DATA   AGE
default-token-qg9qb   kubernetes.io/service-account-token   3      7d18h
docker-credentials    kubernetes.io/dockerconfigjson        1      7d18h
```

If you see a connection or authentication error,
ensure you are using the right kubeconfig from step 3 above.

### Set up a GCP dev environment
1. Start [GCP free trial](https://cloud.google.com/free). Google offers free tier products even after the free trial period is over, and there is no charge as long as the usage stays within the free usage limit. If you still have concerns, feel free to post on Ed, email Xiao, or drop by office hours to discuss.
2. Install [the gcloud CLI](https://cloud.google.com/sdk/docs/install).
3. `gcloud init`
4. Create a VM instance from the base image we provide:
```
gcloud beta compute instances create my-cs426-instance --zone=us-east1-b \
--machine-type=e2-micro --subnet=us-east1 \
--image=https://www.googleapis.com/compute/v1/projects/pro-core-374518/global/images/cs426-dev-spring2023 \
--boot-disk-size=10GB --boot-disk-device-name=my-cs426-instance
```
5. Once the instance creation is complete (takes a minute or so), you can use the `ssh` and `scp` from the gcloud CLI: `gcloud compute ssh root@my-cs426-instance`

This is an ubuntu linux environment that has the above tools installed. Feel free to install additional software in your VM instance if needed.


## Part A. Building, testing, and pushing a Docker image

### A1. Building an image
Docker is a tool for creating, running, and managing containers. It wraps up low-level
APIs like cgroups, chroots, and namespaces into a coherent API. In order to *run* a container
in Docker you must first have an *image*. If you want a more detailed overview of Docker, consider https://docker-curriculum.com/.

Docker images are like a snapshot of a file system -- they include your app and any libraries or configs
needed to run it. Images are layered on top of each other to be built. Typically you start with
some base image (say `ubuntu:latest` to have an image that is like [Ubuntu](https://ubuntu.com/)),
and build layers with operations like building your app and copying it to a well-defined location.

Image builds are specified in a `Dockerfile` as a set of steps. You will be following
the official Docker documentation for Go to create a `Dockerfile` for your VideoRecService.

Open your Lab 1 directory (or copy it to a new spot if you want to snapshot your old work).
Create an empty file called `Dockerfile` in this directory, and follow the steps here:
https://docs.docker.com/language/golang/build-images/

For part **A1**, you'll need the steps:
 1. [Creating a Dockerfile](https://docs.docker.com/language/golang/build-images/#create-a-dockerfile-for-the-application)
 2. [Build the image](https://docs.docker.com/language/golang/build-images/#build-the-image). Don't worry about tagging for now. We will tag your image in **A3**.


#### Extra credit

For this extra credit on this assignment, structure your `Dockerfile` as a multi-stage build for efficiency.
Follow the instructions for [multi-stage builds](https://docs.docker.com/language/golang/build-images/#multi-stage-builds).
For the deploy stage, please use `alpine:latest` as your base image instead of `gcr.io/distroless/base-debian10` in the instruction;
additionally, you do **not** need to switch to `USER nonroot:nonroot`.

#### Build instructions (ARM, Apple M1 / Apple Silicon)

The remote cluster is running on x86_64 machines, which presents challenges if you are not building for the right architecture.
This is easiest if you have an x86_64 machine (easiest on Linux), but
if you are not on a linux/amd64 platform (e.g., Apple sillicon; or if `uname -m` does not output `x86_64`), you should use `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` (instead of `go build`) in your Docker file, and use docker cross-platform builds by doing `docker buildx build --platform=linux/amd64 ...`. Correspondingly, you can do `docker run --platform=linux/amd64 ...`

### A2. Testing your image

If your `docker build` was successful, you should get back a hash for the image (or, if you tagged it in some way, you will have a tag). You can test running your image now with

```
docker run --rm -p 8080:8080 <image-hash>
```

This will expose port `8080` on your machine to port `8080` in the container. You should see your startup logging, but your container won't be able to talk to UserService/VideoService.

#### Optional, if you wish to test a full setup locally
You can make this work by linking the network of the container to your host, then telling
your VideoRecService to use that address instead of localhost:

For Linux:
```
docker run --rm -p 8080:8080 --add-host host.docker.internal:host-gateway <image-hash> /video_rec_service --video-service=host.docker.internal:8082 --use
r-service=host.docker.internal:8081
```

For Docker Desktop (Mac/Windows):
```
docker run --rm -p 8080:8080 <image-hash> /video_rec_service --video-service=host.docker.internal:8082 --use
r-service=host.docker.internal:8081
```
Where `/video_rec_service` is the command to run in the container (which may be different, depending on your Dockerfile, it should be the path of the compiled binary).
You can then run the tools from Lab 1 (like `frontend.go`) and see if they work--you can start the user and video services as normal (with `go run user_service/server/server.go`).

To stop running the container, you can simply `Control-C` it. If you ran with the `--rm` flag like
above it will clean up the container automatically.

You will get a chance to test with Kubernetes in later steps if you don't do this now.


### A3. Pushing your image to a registry

When you previously built your image, it was only stored locally. When deploying your application you
will need to upload your image somewhere so that the cluster can download it and run it. Docker
registries are the canonical way to upload your image to a centralized location.

First you'll need to login to the class registry. Your Docker registry credentials
are already stored for you as a `Secret` object in Kubernetes. You'll need to have
your `kubectl` config setup for this part. You can see the `Secret` object with

```
kubectl get secret docker-credentials
```

Make sure you've done the steps from **Setup** to get your Kubernetes config working.


You can view the full secret with `kubectl get secret docker-credentials -o yaml`. The username/password
are base64 encoded in the `data` field.

To decode it, you can run this bash one-liner:

```
kubectl get secret docker-credentials -o jsonpath="{.data.\\.dockerconfigjson}" | base64 -d
```

The username should be your Net ID, and the password should be a combination of 5-letter words (from [Wordle](https://www.nytimes.com/games/wordle/index.html)).

Use the username and password to login to the Docker registry.

```
docker login registry.cs426.cloud
```

Next, you'll need to tag a build of your image. If you have an image hash that works, you can
run `docker tag <hash> registry.cs426.cloud/<NETID>/video-rec-service:latest`. If you want
to build a new image and tag in one go, use `docker build . -t registry.cs426.cloud/<NETID>/video-rec-service:latest`.

Once you have an image tagged, you can push it to upload to the registry.

```
docker push registry.cs426.cloud/<NETID>/video-rec-service:latest
```

**Note**: The registry storage is shared for the class, so please keep the tags you push to a
single `latest` version.

## Part B. Deploying your service to a Kubernetes cluster

As you've gone over in class, Kubernetes is a distributed scheduler for deploying and managing
containers, service networking, and storage. Kubernetes is fundamentally a _declarative_ API:
you describe a resource that you want to exist, and the Kubernetes control plane eventually
creates or updates that resource.

### B1. Making a deployment
**Deployments** are a Kubernetes resource for deploying a set of containers. All
jobs in Kubernetes are run as **Pods**. Pods are ephemeral instances -- they are removed if
the physical host crashes. Deployments manage a set of Pods and ensure they keep running. If the
Pods die or are removed for any reason  the deployment
will reschedule them. This makes them a common choice for deploying reliable applications.

**Resources** in Kubernetes (e.g., a Deployment) are typically described in [YAML](https://yaml.org/)
files (though you can use a number of formats, including JSON). You'll start deploying your
VideoRecService by creating a `deployment.yml` file.
(If you're unfamiliar with YAML, you can check out the full spec [here](https://yaml.org/spec/1.2.2/). Chapter 2 the language overview might be particularly helpful.)

Deployment specifications generally have two parts: `metadata` about the deployment and the `spec` for the pods it manages, for example:

```
apiVersion: apps/v1
kind: Deployment
metadata:
   ...
spec:
   ...
```

#### metadata
The only important bit you need for metadata is to give
your deployment a `name`. This name can be anything you
wish, but likely should be something like `video-rec` for ease of use. You need **not** include your
NetID in the name, since your deployment will be created
within your own kubernetes namespace.

```
...
metadata:
    name: video-rec
...
```

You may also attach a label to your deployment to help
manage it later, though this is not required.

```
...
metadata:
    name: video-rec
    labels:
        app: video-rec
...
```

This gives `deployment/video-rec` a label of `app=video-rec`, which you could use to manage related resources or deployments (like `kubectl get deployment -l app=video-rec` if you had multiple).

#### spec

Next is the `spec`, the specification for the pods the deployment manages or runs. We can divide this into
three important steps:

1. `replicas` -- the number of pods to run
2. `selector` -- which pods are managed by the deployment
3. `template` -- spec of pods to create

For now, you can set `replicas` to `1` to create a single pod.

```
apiVersion: apps/v1
kind: Deployment
metadata:
   ...
spec:
   replicas: 1
   ...
```

`selector` is used to match pods that are managed by the deployment. For this lab
we'll using a match app label for the `selector` and `template` [\*].

```
...
spec:
    replicas: 1
    selector:
        matchLabels:
            app: video-rec
...
```

The final part is the `template` block, which defines the pods to create.
You can see the documentation for a pod spec here: https://kubernetes.io/docs/concepts/workloads/pods/

```
spec:
    ...
    template:
        metadata:
            labels:
                app: video-rec
```

The final part is to add some containers to the `spec` part of your pod `template`

```
spec:
    ...
    template:
        metadata:
            ...
        spec:
            imagePullSecrets:
                - name: docker-credentials
            containers:
                - name: video-rec
                  image: registry.cs426.cloud/<netid>/<image-name-from-part-A>:latest
                  ports:
                      - containerPort: 8080
                  args: [<args to configure your server>]
                  imagePullPolicy: Always
```

A few things were added here for you:
 - `imagePullSecrets` is essentially a login to the private registry that allows downloading the image you pushed in part A. `docker-credentials` is a Kubernetes object of type [`Secret`](https://kubernetes.io/docs/concepts/configuration/secret/) in your namespace that we provided for you.
 - `imagePullPolicy: Always` means to always try to download your image on startup, even if it is cached. This is not recommended for production, but it means you can just re-push the `:latest` tag and restart your deployment to update, which may help you to debug more quickly in future steps.

You need to expose a port in the `ports` section based on the port your server listens on (see `server.go` for the default which is given above). If you haven't changed the default, you can just
use the above example with `containerPort: 8080`.
You can find the syntax in the official deployment docs, along with other information here: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/


You'll need to pass some arguments in via `args` to configure your service correctly
for the cluster environment. If you already know what to pass here, feel free to do so, otherwise
you will set up the arguments in **B2**.

You can try creating your deployment (and thus pods) now, without passing in any arguments (remove the `args: ` line or make it an empty array).

```
kubectl apply -f deployment.yml
```

The `apply` subcommand will take a YAML spec and update the cluster to the new spec.
This is a universal command that can be used to create new resources or update existing ones.

If you've done everything successfully up to this point, you should see the output (may be a different name, depending on what you used):
```
deployment.apps/video-rec created
```

You can check on the status of the deployment with `kubectl get deployments`. You should see that
there are `0/1` or `1/1` pods ready. If you did everything right, your pod will eventually
be ready (after it downloads the image and starts up). You can check on the pods with
`kubectl get pods`.

**Debugging**: If your pod doesn't start successfully, you can get more debugging information about
your pod with `kubectl describe pods`. You can also view logs with `kubectl logs <pod-name>` where
the pod name is something like `video-rec-abc12` (auto-generated by the deployment, which you can obtain from `kubectl get pods`). Alternatively, given your pod `app` label, use  `kubectl logs -l app=<pod-app-label>`.

**[\*]**: We use the same `label` here for both the selector and template so that the deployment
will manage the pods that it creates. This is the most common way to make a deployment, but the
level of indirection allows for more flexible but complicated setups.

**Discussion**: What do you think will happen if you delete the pod running your service? Note
down your prediction, then run `kubectl delete pod <pod-name>` (you can find the name with `kubectl get pod`) and wait a few seconds, then run `kubectl get pod` again. What happened and why do you think so?

Note this under a heading **B1** in `discussions.md`.

### B2. Configuring your deployment for the cluster

There are three configuration parameters you'll need for running in the cluster
as opposed to running locally:

 1. **Batch size**: the production `UserService` and `VideoService` are running with a max batch size of 50, so you'll need to pass in `--batch-size=50` in your args to match.
 2. **UserService address**: When you're testing locally, the default `UserService` address of `[::1]:8081` (localhost port 8081) is used. When running in the cluster, `UserService` still uses port `8081`, but you'll need to use its DNS hostname to connect as opposed to localhost.
 3. **VideoService address**: Similarly to `UserService`, `VideoService` still runs on port `8082` in the cluster, but you'll need its DNS hostname.

You should piece together the DNS hostname from the following information:
* To mimic production environment, we run `UserService` and `VideoService` for you that acts as a shared microservice for everyone's VideoRecServices. Both `UserService` and `VideoService` expose their interface via a Kubernetes `Service`, and the
control plane will automatically create DNS entries for these services that you can use
to address them. Read the documentation here on `Services` and DNS entries: https://kubernetes.io/docs/concepts/services-networking/service/#dns
* Both `UserService` and `VideoService` are running in the `default` namespace under the service names `user-service` and `video-service` respectively.
* You need to construct an argument
like `--user-service=<dns_hostname>:<port>` to pass to your VideoRecService deployment.

Once you've updated the arguments list in your `deployment.yml` you can
update the deployment in the cluster by running `kubectl apply -f deployment.yml` again. `kubectl logs ...` should show your VideoRecService has successfully refreshed cache since it should be able to talk to the VideoService.

### B3. Exposing the deployment with a service

Just like `UserService` and `VideoService` are exposed as a Kubernetes `Service` resource
so they can be discovered and connected to, you will expose your instance of `VideoRecService`.

Start a new YAML file `service.yml`. You can follow the typical example in the `Service` docs here: https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service

Notably you can name the service whatever you want, but it should likely be something like `video-rec` or `videorecservice` (or some variation). The service name must be a valid DNS label name, so some characters are forbidden. You need not include your net ID or a unique string.

The `selector` used must match the `pods` from your deployment -- you can use the same `app: <pod-app-label>` like in your deployment.

For ports, your service must expose `VideoRecService` **on port 80**. You can have whatever
`targetPort` you want to match the container port (which matches the port the server is listening on in code, default 8080). Kubernetes networking will automatically send traffic on the services cluster IP and port to the pods `targetPort`.

Once you have your `service.yml` ready, you can create the service just like you did the deployment:
```
kubectl apply -f service.yml
```

To verify your service was successfully created:
* `kubectl get services` should show your service.
* `kubectl port-forward services/<your-service-name> 8080:80` will port forward your local port 8080 to a port on the pod (service port 80 which is connected to a port on your pod). You can optionally read more about portforwarding [here](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/#forward-a-local-port-to-a-port-on-the-pod). In a separate terminal, if you start a user service locally (say on port 8081) or comment out the part where frontend talks to the user service, and run `go run cmd/frontend/frontend.go`, you should see the frontend outputs recommendations (though different from the expected output since we run User|VideoServices with a different seed).

### B4. Checking your service with a frontend

If you've done everything right up to now, you should have a healthy deployment running
in your namespace along with a service that exposes this deployment so others can
connect to it. Your VideoRecSerivce now sits as a middle component in the production cluster
to recommend videos!

You can test the whole thing out by visiting the frontend application at
```
https://lab2.cs426.cloud/recommend/$your-k8s-namespace/$your-service-name/$your-net-ID-or-other-user-ID-string
```

**Discussion**: Copy the results into a section **B4** in `discussions.md`.

### B5. Custom resources and load testing

In this part, you'll learn about and use **Custom Resources** in Kubernetes to perform load tests for your VideoRecService.

[Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) allows you to extend the Kubernetes API to define endpoints to store a collection of user-defined objects. They are an extension to the resources provided by kubernetes (such as Pods or Deployments). Their definitions are called Custom Resource Definitions (CRDs). Once defined, CRD objects can be accessed similar to how pods, deployments, and services are accessed, e.g., via `kubect get` or `kubctl apply`.

We use two types of CRDs, `LoadgenRequest` and `LoadgenResult`. In the shared cluster, we've deployed a "load-controller", which watches any `LoadgenRequest` created in any namespace; the controller then sends `GetStats` and `GetTopVideos` requests based on the parameters in the LoadgenRequest resource; after completing the loadgen (or encountering an error), the load-controller writes the result (or error) into a `LoadgenResult` object in the same namespace.

To use the loadgen functionality, first create a YAML file (e.g., `request.yml` to specify the loadgen request.
```
apiVersion: cs426.cloud/v1
kind: LoadgenRequest
metadata:
    name: <name-your-loadgen-request-object>
spec:
    service_name: "<your-video-rec-service-name-from-B3>"
```
Running `kubectl apply -f request.yml` will update the cluster with this new request. You can optionally specify the following parameters beyond the service name. The following are their default values.
```
...
spec:
    service_name: ...
    target_qps: 4
    duration_s: 60
    max_outstanding: 10
```
`max_outstanding` limits how many requests the loadgen controller will send to your VideoRecService without receiving responses or errors. Also note that since loadgen controller is a shared service, it will error out any loadgen requests with `target_qps > 100` or `duration_s > 300`.

To see all available LoadgenResult, use `kubectl get loadgenresults`. To see a particular results, use `kubectl get loadgenresults <name-of-your-loadgen-request> -oyaml` The LoadgenResult may contain an error message if it failed (e.g., if the target_qps is set too high or there's an unexpected crash), in which case feel free to fix the issue and try again.

Other notes:
* You only have create and delete permissions to `LoadgenRequest`s; no updates. I.e., `LoadgenRequests` are immutable, which also means you should use a different name for a new `LoadgenRequest` _every time_.
* LoadgenRequests will be garbage collected within a few minutes of completion.
* You may have 10 different ongoing loadgen requests and at most 20 loadgenresults in your namespace. Please delete the resources you are not actively using. `kubectl delete loadgenresults --all` or `kubectl delete loadgenrequests --all` might come in handy.

**Instructions**
* Delete all but **one** `LoadgenResult` object in your namespace -- that is the one we will grade on. This loadgen run has to have at least one successful request (i.e., not 100% errors) to receive credit.
  * The non-100% error rate is the only expectation on "performance". Latency or QPS are not the focus for this part.
* Include `result.yml` in your submission, which is the output of the command to view this `LoadgenResult` object.


## Part C. Connection management and pooling

In Lab 1, you like issued separate `Dial` calls each time a `GetTopVideos` request hit your service. This was simple but likely inefficient: gRPC allows
a single `ClientConn` to make multiple potentially concurrent multiplexed
RPCs.

It may also be surprising to you that `ClientConns` do not necessarily correspond directly to a TCP connection -- it surprised us as well! `ClientConn`
manages TCP behind the scenes because `gRPC` is built on top of HTTP/2, but
it *also* manages things like keepalives and automatic reconnects for you (and
can even open multiple TCP connections for the same `ClientConn`!).

To improve efficiency (and latency) of your `VideoRecService` you'll now
implement connection reuse.


### C1. Reusing a single connection

To start, we'll reuse a single `ClientConn`. Add some members to your
`VideoRecServiceServer struct` to contain your `ClientConn*`s for `UserService`
and `VideoService`. Alternatively, you can store `UserServiceClient`/`VideoServiceClient`s directly.

You'll need to initialize these in some way. Modify your `MakeVideoRecServiceServer` function to initialize these using `Dial()`
initially.

Note that `Dial` can still fail, just not in the ways you may think if you assume it starts connections. By default `Dial` does not block for connections, it will continue with reconnects in the background, and therefore won't fail even if a connection cannot be made. It can still fail pre-validations for things like invalid addresses or invalid TLS credentials. You can assume that these will be fatal. To handle these errors:

 1. Modify the return of `MakeVideoRecServiceServer` to return `(*VideoRecServiceServer, err)`.
 2. Return `err`s if `grpc.Dial()` fails
 3. Modify `main()` in `server.go` to handle these errors. You can use `log.Fatalf` to end the program in this case, assuming the errors are not transient.

Your `VideoRecServiceServer` should now have permanent clients stored that
can be used. Fix up your `GetTopVideos` handler to re-use these clients/`ClientConn`s for all of your calls instead of re-creating them. You may be able to remove some error handling from your `GetTopVideos` handler too.

If you've already done this step (or similar) from Lab 1 that's great! Move on to the next step.

### C2. Observing the results of a single connection

Now that your service is not repeatedly calling `Dial()` you likely
are using a single TCP connection under the hood to make all of your
outgoing calls to `UserService`/`VideoService`. In most cases this is fine,
but when running in the cluster `UserService`/`VideoService` are effectively
behind a TCP-level (L4) load balancer because the `Pods` running them are
hidden behind a `ClusterIP` from the `Service`.

This means that with only one open connection, your requests will always go
to the same backing `Pod`.  You can visualize this by
checking the traffic to the nodes of the backing services.

Delete your old `loadgenrequest` with `kubectl delete loadgenrequest <name-of-loadgenrequest>`. If you forgot
the name you can find it with `kubectl get loadgenrequest`.

Create a new `loadgenrequest` to send some load
to your `VideoRecService`.

Goto the [Service traffic dashboard](https://monitoring.cs426.cloud/d/sFAdX0oVz/service-traffic-dashboard?orgId=1), logging in with your net ID and the same password you used for the Docker registry previously.

Select your NetID from the `NetID/Namespace` dropdown
at the top left of the dashboard. The top two charts
should show you incoming traffic to the production
`UserService`/`VideoService` pods (there are 2 each) from
your `VideoRecService`.
With only one connection pooled, you should see
that the traffic from your pod only hits one backing pod.


### C3. Reusing multiple connections

In order to spread your load out among the backing instances, you can
create multiple connections. Again modify your `VideoRecServiceServer struct`
and `MakeVideoRecServiceServer` but instead of storing a single `ClientConn*` (or `UserServiceClient`/`VideoServiceClient`), use a slice of them for each service. To initialize these slices:

 1. Add a flag to `server.go` near the other flags like `--port` called `--client-pool-size` as a `flag.Int`. You can set the default to `4`.
 2. Add an `int` for `clientPoolSize` to `VideoRecServiceOptions` in `server_lib.go`, and set its value from the above flag in `main()` in `server.go` like the others.
 3. Initialize the slice with `clientPoolSize` number of connections for each service in `MakeVideoRecServiceServer`, calling `Dial` that many separate times for each service to initialize. If any fail, return the error back from `MakeVideoRecService.`

You will employ a trivial load-balancing strategy to decide which client to use: round robin.
Round robin balancing effectively spreads out the load among the potential clients to use by
choosing the "next" client every time you want to use one.
You can use an [atomic counter](https://pkg.go.dev/sync/atomic) or an int with a lock to keep track of the next index into the
slice when you want to get a client. For pseudo code:

```
// state somewhere
index = 0
// ...
get_client_round_robin(clients []Client) {
    client = clients[index % len(clients)]
    index += 1
    return client
}
```

You may need two counters for your two types of clients to replicate this functionality
for both User and VideoService.

Rebuild and re-push your package to the registry, then run `kubectl rollout restart deploy/video-rec-service` to restart and pickup the changes. Re-run the steps from **C2** to send some load requests to your server and view the
load distribution among the backends in Grafana. **Please leave your
VideoRecService running**

**Discussion**: What does the load distribution look like with a client pool size of 4? What would
you expect to happen if you used 1 client? How about 8? Note this down under section **C3** in
`discussions.md`.

You can update your deployment to set `--client-pool-size=n` test your hypotheses.

# End of Lab 2


# Time logging

Source: from Prof. Stan Eisenstat's CS223/323 courses. Obtained via Prof. James Glenn [here](https://zoo.cs.yale.edu/classes/cs223/f2020/Projects/log.html).

Each lab submission must contain a complete log `time.log`. Your log file should be a plain text file of the general form (that below is mostly fictitious):

```
ESTIMATE of time to complete assignment: 10 hours

      Time     Time
Date  Started  Spent Work completed
----  -------  ----  --------------
8/01  10:15pm  0:45  read assignment and played several games to help me
                     understand the rules.
8/02   9:00am  2:20  wrote functions for determining whether a roll is
                     three of a kind, four of a kind, and all the other
                     lower categories
8/04   4:45pm  1:15  wrote code to create the graph for the components
8/05   7:05pm  2:00  discovered and corrected two logical errors; code now
                     passes all tests except where choice is Yahtzee
8/07  11:00am  1:35  finished debugging; program passes all public tests
               ----
               7:55  TOTAL time spent

I discussed my solution with: Petey Salovey, Biddy Martin, and Biff Linnane
(and watched four episodes of Futurama).

Debugging the graph construction was difficult because the size of the
graph made it impossible to check by hand.  Using asserts helped
tremendously, as did counting the incoming and outgoing edges for
each vertex.  The other major problem was my use of two different variables
in the same function called _score and score.  The last bug ended up being
using one in place of the other; I now realize the danger of having two
variables with names varying only in punctuation -- since they both sound
the same when reading the code back in my head it was not obvious when
I was using the wrong one.
```

Your log MUST contain:
 - your estimate of the time required (made prior to writing any code),
 - the total time you actually spent on the assignment,
 - the names of all others (but not members of the teaching staff) with whom you discussed the assignment for more than 10 minutes, and
 - a brief discussion (100 words MINIMUM) of the major conceptual and coding difficulties that you encountered in developing and debugging the program (and there will always be some).

The estimated and total times should reflect time outside of class.  Submissions
with missing or incomplete logs will be subject to a penalty of 5-10% of the
total grade, and omitting the names of collaborators is a violation of the
academic honesty policy.

To facilitate analysis, the log file MUST the only file submitted whose name contains the string "log" and the estimate / total MUST be on the only line in that file that contains the string "ESTIMATE" / "TOTAL".

# Collaboration policy

## General Statement on Collaboration
TL;DR: Same as [CS323](https://zoo.cs.yale.edu/classes/cs323/current/syllabus.html) for the individual labs (which Labs 0-4 are).

Programming, like composition, is an individual creative process in which you must reach your own understanding of the problem and discover a path to its solution. During this time, discussions with others (including members of the teaching staff) are encouraged. But see the Gilligan's Island Rule below.

However, when the time comes to design the program and write the code, such discussions are no longer appropriate---your solution must be your own personal inspiration (although you may ask members of the teaching staff for help in understanding, designing, writing, and debugging).

Since code reuse is an important part of programming, you may study and/or incorporate published code (e.g., from text books or the Net) in your programs, provided that you give proper attribution in your source code and in your log file and that the bulk of the code submitted is your own. Note: Removing/rewriting comments, renaming functions/variables, or reformatting statements does not convey ownership.

But when you incorporate more than 25 lines of code from a single source, this code (prefaced by a comment identifying the source) must be isolated in a separate file that the rest of your code #include-s or links with. The initial submission of this file should contain only the identifying comment and the original code; revisions may only change types or function/variable names, turn blocks of code into functions, or add comments.

DO NOT UNDER ANY CIRCUMSTANCES COPY SOMEONE ELSE'S CODE OR GIVE A COPY OF YOUR CODE TO SOMEONE ELSE OR OTHERWISE MAKE IT PUBLICLY AVAILABLE---to do so is a clear violation of ethical/academic standards that, when discovered, will be referred to the Executive Committee of Yale College for disciplinary action. Modifying code to conceal copying only compounds the offense.

## The Gilligan's Island Rule

When discussing an assignment with anyone other than a member of the teaching staff, you may write on a board or a piece of paper, but you may not keep any written or electronic record of the discussion. Moreover, you must engage in some mind-numbing activity (e.g., watching an episode of Gilligan's Island) before you work on the assignment again. This will ensure that you can reconstruct what you learned, by yourself, using your own brain. The same rule applies to reading books or studying on-line sources.


