FROM gcr.io/cloud-builders/go:debian
SHELL ["/bin/bash", "-c"]

#RUN git clone https://github.com/mbj4668/pyang.git /workspace/results/pyang@head/pyang
RUN git clone https://github.com/openconfig/oc-pyang /workspace/oc-pyang-repo
RUN git clone https://github.com/robshakir/pyangbind /workspace/pyangbind-repo

RUN apt-get update
RUN apt install -y python3-pip
RUN pip3 install virtualenv
#RUN virtualenv /workspace/pyangvenv

# Install packages for latest pyang venv.
RUN pip3 install wheel && \
        pip3 install pyaml && \
        pip3 install enum34 && \
        pip3 install jinja2 && \
        pip3 install setuptools && \
        pip3 install --no-cache-dir -r /workspace/oc-pyang-repo/requirements.txt && \
        pip3 install --no-cache-dir -r /workspace/pyangbind-repo/requirements.txt
#        pip3 install pyangbind

RUN apt install -y npm && npm install -g npm && npm install -g badge-maker

# Downloading gcloud package
RUN curl https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz > /tmp/google-cloud-sdk.tar.gz

# Installing the package
RUN mkdir -p /usr/local/gcloud \
  && tar -C /usr/local/gcloud -xvf /tmp/google-cloud-sdk.tar.gz \
  && /usr/local/gcloud/google-cloud-sdk/install.sh

# Adding the package path to local
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

