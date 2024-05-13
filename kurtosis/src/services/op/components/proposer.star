def launch(plan, image, files, env, l1, proposer_rpc_port, node_rpc_url):
    proposer_rpc_url = "http://{}:{}".format("0.0.0.0", proposer_rpc_port)
    return plan.add_service(
        name="op-proposer",
        config=ServiceConfig(
            image=image,
            cmd=[
                "op-proposer",
                "--l1-eth-rpc={}".format(l1.rpc_url),
                "--poll-interval=12s",
                "--rpc.addr=0.0.0.0",
                "--rpc.port={}".format(proposer_rpc_port),
                "--rollup-rpc={}".format(node_rpc_url),
                "--l2oo-address=$(jq -r '.L2OutputOracleProxy' contracts-bedrock/deployments/getting-started/l1.json)",
                "--private-key={}".format(env["GS_PROPOSER_PRIVATE_KEY"]),
            ],
            ports={
                "rpc": PortSpec(proposer_rpc_url),
            },
            files={
                "/contracts-bedrock/": files.contracts,
            },
            env_vars=env,
        ),
    )