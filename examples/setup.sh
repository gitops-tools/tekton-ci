#!/bin/sh
kubectl apply -f https://github.com/tektoncd/pipeline/releases/download/v0.10.1/release.yaml
kubectl apply -f https://github.com/tektoncd/triggers/releases/download/v0.3.1/release.yaml
