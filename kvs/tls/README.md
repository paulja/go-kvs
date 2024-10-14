## Important

The domain used in the certificates (when running locally) must also go in the `hosts` file.
The CA cert has to be added to the machine or via gRPC config in code by way of `credentials.TransportCredentials`.
