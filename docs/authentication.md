Authentication between the cloud cluster and the edge clusters works as
following:
  1. on initialization of the cloud components, a self-signed certificate
     for the https server is created
  2. when a new edgecluster resource is created in the cloud cluster, a 
     token is created for authenticating the cluster using the following
     format:
       {random_32byte_hex}:{ca_signed_random_32byte_hex}:{ca_cert_sha256}
  3. the token is then set in configmaps/server-token on the edge cluster

