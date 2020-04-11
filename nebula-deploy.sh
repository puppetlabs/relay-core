# This file contains long-running branch deployment instructions. Once a
# workflow exists for your target environment, configure this file to point your
# branch at that workflow.

# NEBULA_WORKFLOWS[features/iteration-3-rbac]=nebula-deploy-tr-i3-rbac-1
NEBULA_WORKFLOWS[improvements/workflows]=nebula
NEBULA_WORKFLOWS[improvements/cluster-security]=nebula-system-deploy-tr-security
NEBULA_WORKFLOWS[features/PN-151-prometheus-instrumentation]=features-tr-metrics
NEBULA_WORKFLOWS[tasks/pn-576-triggers-crds]=relay-feature-triggers-v2
