PROTO := proto/agent/v1/agent.proto

.PHONY: proto proto-go proto-py sdk-test backend-run frontend-dev

proto: proto-go proto-py

proto-go:
	protoc -I proto --go_out=backend --go_opt=module=backend \
		--go-grpc_out=backend --go-grpc_opt=module=backend $(PROTO)

proto-py:
	mkdir -p sdk/python/omnitask_sdk/gen
	sdk/python/.venv/bin/python -m grpc_tools.protoc -I proto \
		--python_out=sdk/python/omnitask_sdk/gen --grpc_python_out=sdk/python/omnitask_sdk/gen $(PROTO)
	touch sdk/python/omnitask_sdk/gen/__init__.py sdk/python/omnitask_sdk/gen/agent/__init__.py sdk/python/omnitask_sdk/gen/agent/v1/__init__.py
	sed -i 's#^from agent.v1 import agent_pb2 as agent_dot_v1_dot_agent__pb2#from omnitask_sdk.gen.agent.v1 import agent_pb2 as agent_dot_v1_dot_agent__pb2#' sdk/python/omnitask_sdk/gen/agent/v1/agent_pb2_grpc.py

sdk-test:
	cd sdk/python && ./.venv/bin/python -m unittest discover -s tests -t .

backend-run:
	cd backend && nunu run

frontend-dev:
	cd frontend && npm run dev
