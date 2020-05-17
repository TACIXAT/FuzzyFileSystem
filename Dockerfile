FROM ubuntu:20.04

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update -y
RUN apt-get install -y fuse3 wget git

WORKDIR /
RUN wget https://dl.google.com/go/go1.14.linux-amd64.tar.gz -P /tmp
RUN tar xvf /tmp/go1.14.linux-amd64.tar.gz
RUN mkdir /mnt/ffs

WORKDIR /root
RUN echo "export PATH=$PATH:/go/bin" >> .bashrc

WORKDIR /link
RUN /go/bin/go get bazil.org/fuse
RUN git clone https://github.com/TACIXAT/FuzzyFileSystem
ENTRYPOINT ["/bin/bash"]