FROM alpine:3.11
COPY shell-script-csi-driver /bin/
ENTRYPOINT ["/bin/shell-script-csi-driver"]
