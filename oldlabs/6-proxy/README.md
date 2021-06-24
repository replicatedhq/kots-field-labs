Lab 1.6: Proxy
=========================================

In this lab, we'll explore configuring a proxy server in an airgapped environment.

### Instance Overview

As in [Lab 1.5](../lab5-airgap), You will have received the IP of a jump box and the name of an airgapped server.
For example, you may have received:

```text
dx411-dex-lab6-proxy-jump = 104.155.131.205
dx411-dex-lab6-proxy
```

In general, the name of the private server will be the same as the jump box, with the characters `-jump` removed from the suffix.
Put another way, you could construct both instance names programatically as

```shell
${REPLICATED_APP}-lab6-proxy-jump
```

and

```shell
${REPLICATED_APP}-lab6-proxy
```

An HTTP proxy has been provisioned to be shared by all lab participants. 
It will have a dynamic private IP but for simplicity we'll can use the DNS entry 

```
kots-field-labs-squid-proxy
```

### The proxy environment

To start, let's SSH via the jump box and explore our server in the private network.


```shell
export JUMP_BOX_IP=lab6-proxy
export REPLICATED_APP=... # your app slug
ssh -J dex@${JUMP_BOX_IP} dex@${REPLICATED_APP}-lab1-e6-proxy
```

You'll note that egress is not possible by typical means

```shell
curl -v https://kubernetes.io
ping kubernetes.io
```

However, we are able to tunnel out through the proxy server

```shell
curl -x kots-field-labs-squid-proxy:3128 https://kubernetes.io
```

You can use an api.replicated.com endpoint to check the observed egress IP address of your request

```
curl -x kots-field-labs-squid-proxy:3128 https://api.replicated.com/market/v1/echo/ip
```

#### Question 1

Which IP address was printed by the above command?

<details>
  <summary>Answer</summary>
The IP above will be the public IP of the <code>kots-field-labs-squid-proxy</code> server.
</details>


We can also use environment variables to configure a proxy. Let's try setting `HTTP_PROXY`. 

```shell
export HTTP_PROXY=kots-field-labs-squid-proxy:3128
curl https://api.replicated.com/market/v1/echo/ip
```

This command will hang too, we'll explore why below.

#### Question 2

It looks like that didn't work -- what environment variable should we set to enable egress in this case?


<details>
  <summary>Answer</summary>

We can use <code>HTTPS_PROXY</code> in this case since we are making a call with HTTPS and not plain HTTP.

    export HTTPS_PROXY=kots-field-labs-squid-proxy:3128
    curl https://api.replicated.com/market/v1/echo/ip

For more details, explore https://everything.curl.dev/usingcurl/proxies#proxy-environment-variables
> All these proxy environment variable names except http_proxy can also be specified in uppercase, like HTTPS_PROXY.
</details>


### Getting an install script


First we'll get the kURL install script for our channel. From your workstation:

```shell
export REPLICATED_APP=...
export REPLICATED_API_TOKEN=...
replicated channel inspect lab6-proxy
```

Grab the install script from the `EMBEDDED` section.

You can also use the UI to get the install script. 
Ensure you select the right app and channel, and the `Embedded Cluster` option.

<details>
  <summary>Get a Script from the UI</summary>
<img alt="embedded-script" src="img/embedded-script.png">
</details>


### Installing KOTS
Fortunately, KOTS and kURL have built-in support for these types of proxy environments.
There are many ways to do this, the simplest being to set the HTTP_PROXY, HTTPS_PROXY, and NO_PROXY
variables in your shell before runing the kURL install script.
We'll use the environment variable method today, but you can also use a [kurl install-time patch](https://kurl.sh/docs/install-with-kurl/#select-examples-of-using-a-patch-yaml-file).


```shell
export HTTP_PROXY=kots-field-labs-squid-proxy:3128
export HTTPS_PROXY=kots-field-labs-squid-proxy:3128
# use install script we grabbed above, adding a -E flag to sudo
curl -sSL https://k8s.kurl.sh/dx411-dex-lab6-proxy | sudo -E bash
```

Note that you'll need to add `-E` flag to the `sudo` command in order to forward your environment to `bash` process runing under `sudo`.
You can also experiment with only forwarding specific variables as suggested in [this article](https://www.petefreitag.com/item/877.cfm).

> The sudo command has a handy argument -E or --preserve-env which will pass all your environment variables into the sudo environment.
 
 
### What this does
 
When the kURL script detects a proxy configuration in the environment, it will do several things:

- ensure the container runtime (docker or containerd) is configured to pull images via the proxy
- ensure the KOTS admin console is configured to pull app updates and license metadata through the proxy

Once the install skip completes, you can validate this by reviewing the environment variables on 
the `kotsadm` deployment.

### Configuring the instance

As we did in the airgap scenario, we'll open two SSH tunnels to access the admin console and the app.
Run the following on your workstation.

```shell
ssh -NL 8800:${REPLICATED_APP}-lab6-proxy:8800 -L 8888:${REPLICATED_APP}-lab6-proxy:8888 dex@${JUMP_BOX_IP}
```

From here, we can explore a few last things about our environment


### Exploring the install script changes


```shell
kubectl get deploy kotsadm -o json | grep -EA1 'HTTP.*_PROXY'
```

To get `kubectl` working, we'll need to set one more proxy variable, to keep `kubectl` from trying to 
tunnel out through the proxy to talk to `kube-apiserver`.


We'll use ifconfig but you can use whichever preferred method to discover the private IP of your instance.

![ifconfig](img/ifconfig.png)

#### Question

What Kubernetes configuration file could you use to determine the private IP that `kubectl` is using?


<details>
  <summary>Answer</summary>

There are a lot of options here, including `/etc/kubernetes/admin.conf`


    ~$ echo $KUBECONFIG
    /etc/kubernetes/admin.conf

    ~$ head /etc/kubernetes/admin.conf
    apiVersion: v1
    clusters:
    - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1ekNDQWMrZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJeE1EUXhOVEF3TURVMU0xb1hEVE14TURReE16QXdNRFUxTTFvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBUFdrCmZkbHAvSmFkZFpWYnRIWEVqeHpNSGllNGh1UWFvaUE5eEdVZUJ1aFZVejJVaGtzb1VQUmJabDd5WXZud3NiMXYKQXBrdG04N3l3N0ZTSGpHVDQ2SDFWbG1BZmpsMDdXKzc2QStscHlKNDlLVWxtNmI4VzM0OUVWcDJnN2hJNSthNwp4QmRSVzJJQWMrdXh2NitCNllySTdQS04xQWtJeThObTFWNW1hbjhqc2l4Kys5RDdwZDh1OTF5c0kyUXBaWVZQCjFHUEJKcTEvaGtteHE1KzA3bmViQys0dzJKNHR1dldydlFhTklPaUZWa1lLVmhvck1BZVhCOTFOeG1KN1dHZ0wKNGJMQkZBR09KSUV0YVN3WnNLbWVyeTI5ZzFOUnpYcGthcWRYc2U3S0kzOVNPL0JlOXo4YjRJR1pZNXFUWUVBLwowQmpoMXRaMGRaZFF2ZjNmS0xVQ0F3RUFBYU5DTUVBd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZMQlF1bUI2THRJNEo1Z1dqeGp4Q2NmQXpWYVFNQTBHQ1NxR1NJYjMKRFFFQkN3VUFBNElCQVFERGpjM1dkWW4yUC8vaE9Vc1cwN3lyNXNSUEs4RWhpUHZpelVnaHlPaWFadGxaUHJ2UAo3eXJsMVhTTkY3bDltWDgxV01BODBSdnpaak4yY2dHVkpnQzRDYSthcWI1ZmpZNWlJaG43RnU2QzBSYVVZdElTCmdzcEw2Z0VlVVNXdjVpaDlCN0JLS2s5NGZkaytpR2h1K1J6djV1QzNLQnI0blBlNE12TmJrREhFU2NpM2w4Z3QKL2c0VTkzd21xdFV4RlQ3YUI5M3grQ2E3UWpSbnRienNxREJOSkJ3SmkzdlhUdFRJdjA1V3RFK21xNlRnZWZ5dQpOTXBTaUlzcEk4MG5BMjY0SlUrN00rZ3NZMDR2bzFDWEE3bkRFaEM3YjJmaWFybFF1M0hCU09DWkNqQUNVNjVlCjZUMnJDdzdESHp3dlE0ZENUUXd0SDVPWEFCbUVuMHdwcC9UUwotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
      server: https://10.240.0.15:6443

</details>

Once we have this info, we can set NO_PROXY to get kubectl working.



```shell
export NO_PROXY=10.240.0.15
kubectl get pod
```

Now we can try the command above to inspect kotsadm

```shell
kubectl get deploy kotsadm -o json | grep -EA1 'HTTP.*_PROXY'
```

You'll see an entry in the env vars where the kURL script has patched the deployment.

```shell
  "name": "HTTP_PROXY"
  "value": "kots-field-labs-squid-proxy:3128"
```
Congrats! You've completed Exercise 6! [Back To Exercise List](https://github.com/replicatedhq/kots-field-labs/tree/main/labs)



### Additional Exercises

- Test out running a KOTS kots.io support bundle through the proxy
- Explore the [Proxy template functions](https://kots.io/reference/template-functions/static-context/#httpproxy) for passing proxy info into a KOTS application. 

