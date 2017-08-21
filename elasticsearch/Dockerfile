FROM docker.elastic.co/elasticsearch/elasticsearch:5.5.1

# Copy our own config.
ADD elasticsearch.yml /usr/share/elasticsearch/config/
# Give access to our config to elasticsearch user group.
USER root
RUN chown elasticsearch:elasticsearch config/elasticsearch.yml
RUN mkdir -p /var/log/elasticsearch
RUN mkdir -p /var/data/elasticsearch
RUN chown -R elasticsearch:elasticsearch /var/log/elasticsearch
RUN chown -R elasticsearch:elasticsearch /var/data/elasticsearch

# Remove x-pack. You should not do this in prod and definitely not with an
# instance of ES that's reachable online...
RUN bin/elasticsearch-plugin remove x-pack --purge

# Now run as elasticsearch user.
USER elasticsearch
