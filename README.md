# College de France automated audio transcripts
Worker and elasticsearch for automated College de France audio transcripts

[![Build Status](https://travis-ci.org/attwad/cdf.svg?branch=master)](https://travis-ci.org/attwad/cdf)

## Worker

The worker periodically polls datastore for scheduled transcriptions, if any it downloads the mp3 files
from the College de France website, converts them to FLAC, stores them in a Google Storage bucket,
sends a Speech to Text request, stores the transcription in the same storage bucket, and index the transcripts
in an elasticsearch instance running in the same Kubernetes cluster.

A periodic job also runs to compute overall statistics about the transcriptions due to limitations of the datastore
in this regard.

## Elasticsearch

Elasticsearch runs as a single (thus "yellow") master&data node in a Kubernetes cluster, it does full text indexing of
the transcripts using the French analyzer.
