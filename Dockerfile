FROM ubuntu:20.04

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update -y
RUN apt-get install -y libfuse3 wget git

WORKDIR /
RUN wget https://dl.google.com/go/go1.14.linux-amd64.tar.gz -P /tmp
RUN tar xvf /tmp/go1.14.linux-amd64.tar.gz

WORKDIR /root
RUN echo "export PATH=$PATH:/go/bin" >> .bashrc

WORKDIR /link
RUN go get bazil.org/fuse
ENTRYPOINT ["/bin/bash"]

# docker run -it --rm -v"C:\Users\tacixat\prog\FuzzyFileSystem:/link" --cap-add SYS_ADMIN --device /dev/fuse --name ffs ffs/main