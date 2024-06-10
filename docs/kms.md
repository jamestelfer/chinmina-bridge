# Using KMS to sign GitHub JWTs

It is more secure (though more complicated) to provide Chinmina with an AWS KMS key to sign JWTs for GitHub requests.

## Uploading the KMS key

1. [Generate the private key][github-key-generate] for the GitHub application.

2. Check the private key and convert it ready for upload
    - the key spec for your GitHub key _should_ be RSA 2048. To verify that this is
      the case, run `openssl rsa -text -noout -in yourkey.pem` and examine the
      output.
    - convert the GitHub key from PEM to DER format for AWS:

        ```shell
        openssl rsa -inform PEM -outform DER -in ./private-key.pem -out private-key.cer
        ```

3. Follow the [AWS instructions][aws-import-key-material] for importing the
   application private key into GitHub. This includes creating an RSA 2048 key
   of type "EXTERNAL", encrypting the key material according to the instructions
   and uploading it.

4. Create an alias for the KMS key to allow for easy [manual key
   rotation][aws-manual-key-rotation].

   > [!IMPORTANT]
   > A key alias is essential to allow for key rotation. Unless you're stopped
   > by environmental policy, use the alias. The key will be able to be rotated
   > without any service downtime.

5. Ensure that the key policy has a statement allowing Chinmina to access the key. The specified role should be the role that the Chinmina process has access to at runtime.

    ```json
    {
        "Sid": "Allow Chinmina to sign using the key",
        "Effect": "Allow",
        "Principal": {
            "AWS": [
                "arn:aws:iam::226140413739:role/full-task-role-name"
            ]
        },
        "Action": [
            "kms:Sign"
        ],
        "Resource": "*"
    }
    ```

    > [!IMPORTANT]
    > Chinmina does not assume a role to access the key. It assumes valid
    > credentials are present for the AWS SDK to use.

## Configuring the Chinmina service

1. Set the environment variable `GITHUB_APP_PRIVATE_KEY_ARN` to the ARN of the **alias** that has just been created.

2. Update IAM for your key.
    1. Key resource policy
    2. Alias policy?
    3. IAM policy for Chinmina process (i.e. the AWS role available to Chinmina
       when it runs)

[github-key-generate]: https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/managing-private-keys-for-github-apps#generating-private-keys
[aws-import-key-material]: https://docs.aws.amazon.com/kms/latest/developerguide/importing-keys.html
[aws-manual-key-rotation]: https://docs.aws.amazon.com/kms/latest/developerguide/rotate-keys.html#rotate-keys-manually
