FROM gcr.io/cloud-builders/go:debian
SHELL ["/bin/bash", "-c"]

#RUN git clone https://github.com/mbj4668/pyang.git /workspace/results/pyang-head/pyang
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