FROM alpine:latest

# Add a test label that we'll try to remove
LABEL quay.expires-after="2024-01-01"
LABEL test.label="should-be-removed"
LABEL maintainer="bcook@redhat.com"

# Simple hello world
RUN echo "Hello, World!" > /message.txt

CMD ["cat", "/message.txt"] 