SERVICES := frontend product-catalog cart checkout payment order loadgen
CLUSTER  := webinar-demo
TAG      := dev

.PHONY: tidy build import cluster-up cluster-down deploy undeploy up wait-ready down logs reload status psql monitoring-install monitoring-uninstall monitoring-status sm-probe-install sm-probe-uninstall sm-probe-status

tidy:
	go mod tidy

build: tidy
	@for s in $(SERVICES); do \
		echo ">> building $$s"; \
		docker build --build-arg SERVICE=$$s -t webinar-demo/$$s:$(TAG) . || exit 1; \
	done

import:
	@for s in $(SERVICES); do \
		echo ">> importing $$s"; \
		k3d image import webinar-demo/$$s:$(TAG) -c $(CLUSTER) --mode direct || exit 1; \
	done

cluster-up:
	k3d cluster create -c deploy/k3d/cluster.yaml

cluster-down:
	k3d cluster delete $(CLUSTER)

deploy:
	kubectl apply -f deploy/k8s/

undeploy:
	kubectl delete -f deploy/k8s/ --ignore-not-found

wait-ready:
	kubectl -n shop wait --for=condition=available --timeout=300s deployment --all

up: cluster-up build import deploy wait-ready

down:
	-$(MAKE) undeploy
	-$(MAKE) cluster-down

logs:
	@test -n "$(SVC)" || (echo "usage: make logs SVC=<service>" && exit 1)
	kubectl -n shop logs deploy/$(SVC) --tail=100 -f

reload:
	@test -n "$(SVC)" || (echo "usage: make reload SVC=<service>" && exit 1)
	docker build --build-arg SERVICE=$(SVC) -t webinar-demo/$(SVC):$(TAG) .
	k3d image import webinar-demo/$(SVC):$(TAG) -c $(CLUSTER) --mode direct
	kubectl -n shop rollout restart deploy/$(SVC)

status:
	kubectl -n shop get pods,svc,ingress

psql:
	kubectl -n shop exec -it deploy/postgres -- psql -U postgres

monitoring-install:
	@test -f .env || (echo "missing .env (see .env.example)" && exit 1)
	@set -a; . ./.env; set +a; \
	helm repo add grafana https://grafana.github.io/helm-charts >/dev/null 2>&1 || true; \
	helm repo update grafana >/dev/null; \
	helm upgrade --install grafana-cloud -n monitoring --create-namespace \
		grafana/grafana-cloud-onboarding \
		--set "cluster.name=$$GRAFANA_CLOUD_CLUSTER_NAME" \
		--set "grafanaCloud.fleetManagement.auth.username=$$GRAFANA_CLOUD_FM_USERNAME" \
		--set "grafanaCloud.fleetManagement.auth.password=$$GRAFANA_CLOUD_FM_PASSWORD" \
		--set "grafanaCloud.fleetManagement.url=$$GRAFANA_CLOUD_FM_URL" \
		--wait

monitoring-uninstall:
	helm uninstall grafana-cloud -n monitoring --ignore-not-found
	kubectl delete namespace monitoring --ignore-not-found

monitoring-status:
	@kubectl -n monitoring get pods,svc 2>&1

sm-probe-install:
	@test -f .env || (echo "missing .env (see .env.example)" && exit 1)
	@set -a; . ./.env; set +a; \
	test -n "$$SM_API_TOKEN" || (echo "SM_API_TOKEN not set in .env" && exit 1); \
	test -n "$$SM_API_SERVER" || (echo "SM_API_SERVER not set in .env" && exit 1); \
	kubectl apply -f deploy/sm-probe/namespace.yaml; \
	kubectl -n synthetic-monitoring create secret generic sm-agent-1 \
		--from-literal=api-token="$$SM_API_TOKEN" \
		--from-literal=api-server="$$SM_API_SERVER" \
		--dry-run=client -o yaml | kubectl apply -f -; \
	kubectl apply -f deploy/sm-probe/deployment.yaml; \
	kubectl -n synthetic-monitoring rollout status deploy/sm-agent-1 --timeout=120s

sm-probe-uninstall:
	kubectl delete -f deploy/sm-probe/deployment.yaml --ignore-not-found
	kubectl -n synthetic-monitoring delete secret sm-agent-1 --ignore-not-found
	kubectl delete -f deploy/sm-probe/namespace.yaml --ignore-not-found

sm-probe-status:
	@kubectl -n synthetic-monitoring get pods,svc,deploy 2>&1
	@echo "---"
	@kubectl -n synthetic-monitoring logs deploy/sm-agent-1 --tail=20 2>&1 || true
