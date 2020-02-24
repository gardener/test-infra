FROM golang:1.13-alpine3.11

ENV HELM_TILLER_VERSION=v2.13.0
ENV KUBECTL_VERSION=v1.14.0

RUN  \
  apk update \
  && apk add \
    apache2-utils \
    coreutils \
    bash \
    binutils \
    bind-tools \
    build-base \
    curl \
    file \
    gcc \
    git \
    jq \
    libc-dev \
    libev-dev \
    libffi-dev \
    openssh \
    openssl \
    openssl-dev \
    python2 \
    python3 \
    python3-dev \
    vim \
    wget \
    grep \
    findutils \
    rsync \
    bc \
    linux-headers \
    libxml2-utils \
  && pip3 install --upgrade pip \
    gardener-cicd-cli \
    gardener-cicd-libs \
    mako \
    semver \
    xmljson \
    xmltodict \
  && ln -s /usr/bin/cli.py /cc/utils/cli.py \
  && curl -Lo /bin/kubectl \
    https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
  && chmod +x /bin/kubectl \
  && curl -L \
    https://kubernetes-helm.storage.googleapis.com/helm-${HELM_TILLER_VERSION}-linux-amd64.tar.gz \
    | tar xz -C /bin --strip=1 \
  && chmod +x /bin/helm \
  && curl -Lo /bin/yaml2json \
    https://github.com/bronze1man/yaml2json/releases/download/v1.2/yaml2json_linux_amd64 \
  && chmod +x /bin/yaml2json \
  && curl -Lo /bin/cfssl \
    https://pkg.cfssl.org/R1.2/cfssl_linux-amd64 \
  && chmod +x /bin/cfssl \
  && curl -Lo /bin/cfssljson \
    https://pkg.cfssl.org/R1.2/cfssljson_linux-amd64 \
  && chmod +x /bin/cfssljson \
  &&  curl http://aia.pki.co.sap.com/aia/SAP%20Global%20Root%20CA.crt -o \
    /usr/local/share/ca-certificates/SAP_Global_Root_CA.crt \
  && curl http://aia.pki.co.sap.com/aia/SAPNetCA_G2.crt -o \
      /usr/local/share/ca-certificates/SAPNetCA_G2.crt \
  && curl http://aia.pki.co.sap.com/aia/SAP%20Global%20Sub%20CA%2002.crt -o \
      /usr/local/share/ca-certificates/SAP_Global_Sub_CA_02.crt \
  && curl http://aia.pki.co.sap.com/aia/SAP%20Global%20Sub%20CA%2004.crt -o \
      /usr/local/share/ca-certificates/SAP_Global_Sub_CA_04.crt \
  && curl http://aia.pki.co.sap.com/aia/SAP%20Global%20Sub%20CA%2005.crt -o \
      /usr/local/share/ca-certificates/SAP_Global_Sub_CA_05.crt \
  && update-ca-certificates

ENV PATH /cc/utils/bin:$PATH
