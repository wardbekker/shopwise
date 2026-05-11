SERVICES := frontend product-catalog cart checkout payment order loadgen
CLUSTER  := webinar-demo
TAG      := dev

.PHONY: tidy build import cluster-up cluster-down deploy undeploy up wait-ready down logs reload status psql

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
