
Example to update mongodb version:

```bash
export VERSION="2.4.14"
mkdir -p ${VERSION}
sed 's/\[% version %\]/2.4.14/' Dockerfile_template > ${VERSION}/Dockerfile
```