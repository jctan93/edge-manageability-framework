# SPDX-FileCopyrightText: 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0


locals {
  lb_sg_ids = {
    "traefik" : {
      port              = 8443,
      security_group_id = var.traefik_sg_id
    },
    "traefik2" : {
      port              = 443,
      security_group_id = var.traefik2_sg_id
    },
    "argocd" : {
      port              = 8080,
      security_group_id = var.argocd_sg_id
    },
    "gitea" : {
      port              = 3000,
      security_group_id = var.argocd_sg_id
    }
  }
}

resource "aws_vpc_security_group_ingress_rule" "node_sg_rule" {
  for_each                     = local.lb_sg_ids
  security_group_id            = aws_security_group.eks_cluster.id
  referenced_security_group_id = each.value.security_group_id
  from_port                    = each.value.port
  to_port                      = each.value.port
  ip_protocol                  = "tcp"
  description                  = "From sg ${each.value.security_group_id} to eks node port ${each.value.port}"
}
