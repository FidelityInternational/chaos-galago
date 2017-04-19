#! /bin/bash

set -e
set -u

echo "Logging into CF..."
cf api "https://api.$CF_SYS_DOMAIN" --skip-ssl-validation
cf auth "$CF_DEPLOY_USERNAME" "$CF_DEPLOY_PASSWORD"
echo "Creating Org chaos-galago..."
cf create-org chaos-galago
echo "Targetting Org chaos-galago..."
cf target -o chaos-galago
echo "Creating Space chaos-galago..."
cf create-space chaos-galago
echo "Targetting Space chaos-galago..."
cf target -s chaos-galago
if [[ $# -gt 0 && $1 != "--managed-db" ]]; then
  echo "Setting up services..."
  if [[ "$(cf service chaos-galago-db || true)" == *"FAILED"* ]] ; then
    echo "Creating service chaos-galago-db..."
    cf cups chaos-galago-db -p "{\"database\":\"$DB_NAME\",\"host\":\"$DB_HOST\",\"port\":\"$DB_PORT\",\"username\":\"$DB_USERNAME\",\"password\":\"$DB_PASSWORD\"}"
  else
    echo "chaos-galago-db already exists..."
    echo "updating chaos-galago-db..."
    cf unbind-service chaos-galago-broker chaos-galago-db
    cf unbind-service chaos-galago-processor chaos-galago-db
    cf uups chaos-galago-db -p "{\"database\":\"$DB_NAME\",\"host\":\"$DB_HOST\",\"port\":\"$DB_PORT\",\"username\":\"$DB_USERNAME\",\"password\":\"$DB_PASSWORD\"}"
  fi
fi
if [[ "$(cf service cf-service || true)" == *"FAILED"* ]] ; then
  echo "Creating service cf-service..."
  cf cups cf-service -p "{\"domain\":\"$CF_SYS_DOMAIN\",\"skipsslvalidation\":$CF_SKIPSSLVALIDATION,\"username\":\"$CF_USERNAME\",\"password\":\"$CF_PASSWORD\"}"
else
  echo "cf-service already exists..."
  echo "updating cf-service..."
  cf unbind-service chaos-galago-processor cf-service
  cf uups cf-service -p "{\"domain\":\"$CF_SYS_DOMAIN\",\"skipsslvalidation\":$CF_SKIPSSLVALIDATION,\"username\":\"$CF_USERNAME\",\"password\":\"$CF_PASSWORD\"}"
fi
echo "Deploying apps..."
if [[ "$(cf app chaos-galago-broker || true)" == *"FAILED"* ]] ; then
  cf push
else
  echo "Zero downtime deploying chaos-galago-broker..."
  domain="$CF_SYS_DOMAIN"
  cf push -f manifest-green.yml
  cf map-route chaos-galago-broker-green "$domain" -n chaos-galago-broker
  cf delete chaos-galago-broker -f
  cf rename chaos-galago-broker-green chaos-galago-broker
  cf unmap-route chaos-galago-broker "$domain" -n chaos-galago-broker-green
  cf push chaos-galago-processor
fi
deployed_domain=$(cf app chaos-galago-broker | grep urls | cut -d":" -f2 | xargs)
echo "Adding as CF Service Broker..."
deployed_domain=$(cf app chaos-galago-broker | grep urls | cut -d":" -f2 | xargs)
if [[ "$(cf create-service-broker chaos-galago admin not_secured "https://$deployed_domain" || true)" == *"FAILED"* ]] ; then
  cf update-service-broker chaos-galago admin not_secured "https://$deployed_domain"
fi
echo "Adding CF Service Access..."
cf enable-service-access chaos-galago
