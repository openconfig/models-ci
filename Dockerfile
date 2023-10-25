# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang
SHELL ["/bin/bash", "-c"]

#RUN git clone https://github.com/mbj4668/pyang.git /workspace/results/pyang@head/pyang
RUN git clone https://github.com/openconfig/oc-pyang /workspace/oc-pyang-repo
RUN git clone https://github.com/robshakir/pyangbind /workspace/pyangbind-repo

RUN apt-get update
RUN apt install -y python3-pip
RUN apt install -y virtualenv
#RUN virtualenv /workspace/pyangvenv

# Install packages for latest pyang venv.
RUN apt install python3-wheel && \
        pip3 install pyaml enum34 --break-system-packages && \
        apt install -y python3-jinja2 && \
        apt install -y python3-setuptools && \
        pip3 install --no-cache-dir --break-system-packages -r /workspace/oc-pyang-repo/requirements.txt && \
        pip3 install --no-cache-dir --break-system-packages -r /workspace/pyangbind-repo/requirements.txt

RUN apt install -y npm
#RUN npm install -g npm
RUN npm install -g badge-maker

# Downloading gcloud package
RUN curl https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz > /tmp/google-cloud-sdk.tar.gz

ENV CLOUDSDK_PYTHON /usr/bin/python3

# Installing the package
RUN mkdir -p /usr/local/gcloud \
  && tar -C /usr/local/gcloud -xvf /tmp/google-cloud-sdk.tar.gz \
  && /usr/local/gcloud/google-cloud-sdk/install.sh

# Adding the package path to local
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

