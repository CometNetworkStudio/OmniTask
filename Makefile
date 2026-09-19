PROTO := proto/agent/v1/agent.proto

.PHONY: proto proto-go proto-py agent-test backend-run frontend-dev

proto: proto-go proto-py

proto-go:
	protoc -I proto --go_out=backend --go_opt=module=backend \
		--go-grpc_out=backend --go-grpc_opt=module=backend $(PROTO)

proto-py:
	mkdir -p agent/gen
	agent/.venv/bin/python -m grpc_tools.protoc -I proto \
		--python_out=agent/gen --grpc_python_out=agent/gen $(PROTO)
	touch agent/gen/__init__.py agent/gen/agent/__init__.py agent/gen/agent/v1/__init__.py

agent-test:
	cd agent && ./.venv/bin/python -m unittest discover -s tests -t .
	cd agent && ./.venv/bin/python test_smoke.py

backend-run:
	cd backend && nunu run

frontend-dev:
	cd frontend && npm run dev
