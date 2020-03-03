<a href="https://www.sparkpost.com"><img alt="SparkPost, Inc." src="doc-img/sp-pmta-logo.png" width="200px"/></a>

[Sign up](https://app.sparkpost.com/join?plan=free-0817?src=Social%20Media&sfdcid=70160000000pqBb&pc=GitHubSignUp&utm_source=github&utm_medium=social-media&utm_campaign=github&utm_content=sign-up) for a SparkPost account and visit our [Developer Hub](https://developers.sparkpost.com) for even more content.

# sparkypmtatracking
[![Build Status](https://travis-ci.org/tuck1s/sparkypmtatracking.svg?branch=master)](https://travis-ci.org/tuck1s/sparkypmtatracking)
[![Coverage Status](https://coveralls.io/repos/github/tuck1s/sparkypmtatracking/badge.svg?branch=master)](https://coveralls.io/github/tuck1s/sparkypmtatracking?branch=master)

Open and click tracking modules for PMTA and SparkPost Signals:

<img src="doc-img/open_click_tracking_for_signals_for_powermta.svg"/>

|app (link)|description|
|---|---|
|[feeder](cmd/feeder/README.md)|Takes open & click events, adds message attributes from Redis where found, and feeds them to the SparkPost Signals [Ingest API](https://developers.sparkpost.com/api/events-ingest/)|
|[tracker](cmd/tracker/README.md)|Web service that decodes client email opens and clicks|
|[acct_etl](cmd/acct_etl/README.md)|Extract, transform, load piped PMTA custom accounting stream message attributes into Redis|
|[wrapper](cmd/wrapper/README.md)|SMTP proxy service that adds engagement information to email
|[linktool](cmd/linktool/README.md)|Command-line tool to encode and decode link URLs|

Click above links for command-specific README.

## Pre-requisites
- Git & Golang - installation tips [here](#installing-git-golang-on-your-host)
- Redis - installation tips [here](#installing-redis-on-your-host)
- NGINX - installation tips [here](#installing-and-configuring-nginx-proxy)


# Build project from source
First, check you have the the [Pre-requisites](#pre-requisites). If you don't have `$GOPATH` set already, do:
```
cd ~
mkdir go
export GOPATH=/home/ec2-user/go # change this to suit the directory you just made
```

Get this project (and its dependent libraries), and build them in place

```
go get github.com/tuck1s/sparkypmtatracking

```

Run the `./build.sh` script included in the project, to build each app.

# Run
Script [start.sh](start.sh) is provided as a starting point for you to customise, along with an example [cronfile](cronfile) that can be used to start services on boot:

```
crontab cronfile
```
or `crontab -e` then paste in cronfile contents.

# CI code tests
The project includes built-in tests as per usual Go / Travis CI / Coveralls practices.

---

# Pre-requisites

## Git, Golang
Your package manager should provide installation for these, e.g.
```
sudo yum install -y git go
```

## Redis on Amazon Linux
```
sudo amazon-linux-extras install epel
sudo yum install -y redis
sudo service redis start
```
This project assumes the usual port `6379` on your host. Check you now have `redis` installed and working.
```
redis-cli --version
```
you should see `redis-cli 5.0.5` or similar
```
redis-cli PING
```
you should see `PONG`.

## NGINX
This can be used to protect your open/click tracking server. The [example config file](etc/nginx/conf/server_example.conf) in this project uses the following Nginx features/modules:
- http-ssl
- http-v2
- headers-more

### NGINX yum/EPEL/webtatic install on Amazon Linux
If you have access to the EPEL and Webtatic repos on your platform, you can use `yum`-based install to get Nginx with added modules:
```
sudo yum update -y
sudo amazon-linux-extras install epel
wget http://repo.webtatic.com/yum/el7/x86_64/RPMS/webtatic-release-7-3.noarch.rpm
sudo rpm -Uvh webtatic-release-7-3.noarch.rpm
sudo yum --enablerepo=webtatic install nginx1w nginx1w-module-headers-more
sudo service nginx start
nginx -V
```

### dhparam
As per the article referred to in the example .conf file, the .conf file expects DH params set up. You can create these with `openssl` and keep them in the usual place for certs. Needs `sudo` to write to this directory.
```
sudo openssl dhparam 2048 -out /etc/pki/tls/certs/dhparam.pem
```

### Standard ports
If you wish to use standard ports (80, 443) for tracking:
- Check the main config file `nginx.conf` is not serving ordinary files by default on those ports.
- If it is, you may need to change or remove the existing `server { .. }` stanza.

Check the endpoint is active from another host, using `curl` as above, but using your external host address and port number.

### Alternative to yum install: source-based install
The standard Nginx version available via `yum` does not have all needed features. You can build from source, providing you have `gcc` and `git` installed.
```
sudo yum install -y gcc git # pre-requisites

wget http://nginx.org/download/nginx-1.16.1.tar.gz
tar -xzvf nginx-1.16.1.tar.gz
wget https://github.com/openresty/headers-more-nginx-module/archive/v0.33.tar.gz
tar -xzvf v0.33.tar.gz
git clone https://github.com/openssl/openssl.git
cd openssl
git branch -a
# Choose the following as I found 1_1_1 gave problems
git checkout remotes/origin/OpenSSL_1_0_2-stable
cd ../nginx-1.16.1
./configure --prefix=/opt/nginx --with-http_ssl_module --with-openssl=../openssl --add-module=../headers-more-nginx-module-0.33  --with-http_v2_module
make
sudo make install
```
This places the freshly built code into `/opt/nginx/`.

Start, stop and reload config as follows:

```
# start
sudo /opt/nginx/sbin/nginx

# stop
sudo /opt/nginx/sbin/nginx -s stop

# reload config
sudo /opt/nginx/sbin/nginx -s reload
```

Ensure nginx starts on boot. There are various ways to do this, the simplest is to append a start command in the file `/etc/rc.local`.