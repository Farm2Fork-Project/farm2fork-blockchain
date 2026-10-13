# EC2 deployment

Merge a tested pull request into **main** to deploy. The workflow runs tests, publishes an immutable `deploy-FULL_SHA` release, then waits for EC2 to confirm activation. PR workflows cannot publish releases. Runtime secrets and model weights stay on EC2; no production `.env` is needed in Actions.

The EC2 deployment agent polls main approximately every minute. Docker manages process restarts. A host lock serializes changes across backend, AI and blockchain. Failed releases are visible in the Actions activation job.

The full operational and recovery runbook is [backend deployment documentation](https://github.com/Farm2Fork-Project/farm2fork-backend/blob/main/docs/ec2-deployment.md). Use `sudo f2fctl status`, `check`, `logs`, `backup`, or `retry COMPONENT` on EC2.

Chaincode upgrades retain ledger state and use the committed sequence plus one. Unchanged chaincode source only verifies the ledger. Never run network-down/reset or remove production volumes. Recovery redeploys the previous package at a higher sequence.
