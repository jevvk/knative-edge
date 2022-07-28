Authentication between the cloud cluster and the edge clusters works as
following:
  1. on initialization of the cloud components, a self-signed certificate
     for the https server is created
  2. when a new edgecluster resource is created in the cloud cluster, a 
     token is created for authenticating the cluster using the following
     format:
       {random_32byte_hex}:{signed_random_32byte_hex}:{ca_cert_sha256}
  3. the token is then set in configmap/server-token on the edge cluster
  4. the edge cluster connects to the https server and checks that the CA
     signature matches the on in configmap/server-token
  5. the server accepts the connection only if the token is known (i.e.
     it belongs to the edgecluster resource), the signature of the token
     is valid (signed using the CA private key), and the CA signature
     matches (likely not necessary for the server)

