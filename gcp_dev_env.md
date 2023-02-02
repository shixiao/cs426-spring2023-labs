# Set up a Google Cloud Platform (GCP) dev environment
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
