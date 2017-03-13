#!/bin/bash
version=$(date +'%Y%m%dat%H%M')
goapp deploy -version=$version -application=waveguide-160005 app.yaml
