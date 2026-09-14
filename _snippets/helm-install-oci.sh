# Export the default values to customize your installation
helm show values oci://ghcr.io/consize-oss/charts/consize > values.yaml

# Install using your customized values
helm install consize oci://ghcr.io/consize-oss/charts/consize \
  --version 0.2.0 \
  --namespace consize-system \
  --create-namespace \
  -f values.yaml
