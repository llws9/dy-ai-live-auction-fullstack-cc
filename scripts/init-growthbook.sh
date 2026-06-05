#!/bin/bash

set -euo pipefail

: "${GROWTHBOOK_API_URL:=http://localhost:3200}"
: "${GROWTHBOOK_MONGO_CONTAINER:=growthbook-mongo}"
: "${GROWTHBOOK_DB:=growthbook}"
: "${GROWTHBOOK_ADMIN_EMAIL:=growthbook-admin@local.test}"
: "${GROWTHBOOK_ADMIN_NAME:=Local Admin}"
: "${GROWTHBOOK_ADMIN_PASSWORD:=Growthbook#2026!}"
: "${GROWTHBOOK_ORG_NAME:=DY Auction Local}"
: "${GROWTHBOOK_CLIENT_KEY:=dev-client-key}"
: "${GROWTHBOOK_SDK_CONNECTION_KEY:=sdk-6u0ZsAdh7mYpvLw}"
: "${FEATURE_KEY:=live-start-popup-visibility}"
: "${PROJECT_FALLBACK_ID:=prj_dy_auction_local}"
: "${USER_FALLBACK_ID:=u_dy_auction_local_admin}"
: "${ORG_FALLBACK_ID:=org_dy_auction_local}"

require_cmd() {
  local cmd=$1
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "错误: 缺少命令 $cmd" >&2
    exit 1
  fi
}

mongo_eval() {
  local script
  script="$(cat)"
  docker exec "$GROWTHBOOK_MONGO_CONTAINER" mongosh --quiet "$GROWTHBOOK_DB" --eval "$script"
}

mongo_scalar() {
  local expression=$1
  mongo_eval <<JS
const result = $expression;
if (result !== null && result !== undefined) {
  print(result);
}
JS
}

require_growthbook_ready() {
  docker inspect "$GROWTHBOOK_MONGO_CONTAINER" >/dev/null 2>&1 || {
    echo "错误: 未找到 Mongo 容器 $GROWTHBOOK_MONGO_CONTAINER" >&2
    exit 1
  }

  curl -fsS "$GROWTHBOOK_API_URL" >/dev/null || {
    echo "错误: GrowthBook API 不可访问: $GROWTHBOOK_API_URL" >&2
    exit 1
  }
}

ensure_growthbook_first_user() {
  local org_count
  org_count="$(mongo_scalar 'db.organizations.countDocuments()')"
  if [[ "${org_count:-0}" != "0" ]]; then
    return
  fi

  echo "初始化 GrowthBook 首个管理员和组织 ..."
  curl -fsS -X POST "$GROWTHBOOK_API_URL/auth/firsttime" \
    -H 'Content-Type: application/json' \
    -d "$(jq -n \
      --arg email "$GROWTHBOOK_ADMIN_EMAIL" \
      --arg name "$GROWTHBOOK_ADMIN_NAME" \
      --arg password "$GROWTHBOOK_ADMIN_PASSWORD" \
      --arg companyname "$GROWTHBOOK_ORG_NAME" \
      '{email:$email,name:$name,password:$password,companyname:$companyname}')" \
    >/dev/null
}

ensure_growthbook_records() {
  local org_id
  local user_id
  local project_id

  org_id="$(mongo_scalar "db.organizations.findOne({}, {id: 1})?.id")"
  user_id="$(mongo_scalar "db.users.findOne({email: '$GROWTHBOOK_ADMIN_EMAIL'}, {id: 1})?.id || db.users.findOne({}, {id: 1})?.id")"
  project_id="$(mongo_scalar "db.projects.findOne({}, {id: 1})?.id")"

  org_id="${org_id:-$ORG_FALLBACK_ID}"
  user_id="${user_id:-$USER_FALLBACK_ID}"
  project_id="${project_id:-$PROJECT_FALLBACK_ID}"

  mongo_eval >/dev/null <<JS
const now = new Date();
const orgId = "$org_id";
const userId = "$user_id";
const projectId = "$project_id";
const adminEmail = "$GROWTHBOOK_ADMIN_EMAIL";
const adminName = "$GROWTHBOOK_ADMIN_NAME";
const orgName = "$GROWTHBOOK_ORG_NAME";
const clientKey = "$GROWTHBOOK_CLIENT_KEY";
const sdkConnectionKey = "$GROWTHBOOK_SDK_CONNECTION_KEY";
const featureKey = "$FEATURE_KEY";

db.organizations.updateOne(
  {id: orgId},
  {
    \$setOnInsert: {
      id: orgId,
      dateCreated: now,
      members: [{id: userId, role: "admin"}],
      invites: [],
      settings: {}
    },
    \$set: {name: orgName, ownerEmail: adminEmail, dateUpdated: now}
  },
  {upsert: true}
);

db.users.updateOne(
  {id: userId},
  {
    \$setOnInsert: {
      id: userId,
      dateCreated: now
    },
    \$set: {email: adminEmail, name: adminName, superAdmin: true, dateUpdated: now}
  },
  {upsert: true}
);

db.projects.updateOne(
  {id: projectId},
  {
    \$setOnInsert: {
      id: projectId,
      dateCreated: now
    },
    \$set: {organization: orgId, name: "My First Project", dateUpdated: now}
  },
  {upsert: true}
);

db.apikeys.updateOne(
  {key: clientKey},
  {
    \$setOnInsert: {
      id: "key_devclientkeylocal",
      dateCreated: now
    },
    \$set: {
      organization: orgId,
      dateUpdated: now,
      key: clientKey,
      secret: false,
      description: "Local publishable key for H5/admin GrowthBook defaults",
      environment: "production",
      project: projectId,
      encryptSDK: false,
      encryptionKey: "",
      disabled: false,
      userId: "",
      role: "",
      limitAccessByEnvironment: false,
      environments: [],
      projectRoles: [],
      lastUsed: null
    }
  },
  {upsert: true}
);

db.sdkconnections.updateOne(
  {key: sdkConnectionKey},
  {
    \$setOnInsert: {
      id: "sdk_dy_auction_local_react",
      dateCreated: now
    },
    \$set: {
      name: "React SDK Connection",
      key: sdkConnectionKey,
      organization: orgId,
      environment: "production",
      projects: [projectId],
      language: "react",
      encryptPayload: false,
      includeDraftExperiments: true,
      includeExperimentNames: true,
      includeVisualExperiments: false,
      includeRedirectExperiments: false,
      includeRuleIds: true,
      includeProjectIdInMetadata: false,
      includeCustomFieldsInMetadata: false,
      allowedCustomFieldsInMetadata: [],
      includeTagsInMetadata: false,
      dateUpdated: now
    }
  },
  {upsert: true}
);

db.features.updateOne(
  {id: featureKey},
  {
    \$setOnInsert: {
      id: featureKey,
      dateCreated: now
    },
    \$set: {
      organization: orgId,
      project: projectId,
      owner: userId,
      description: "Controls whether the live start popup is shown as the first AB experiment.",
      valueType: "boolean",
      defaultValue: "false",
      archived: false,
      dateUpdated: now,
      environmentSettings: {
        production: {enabled: true}
      },
      rules: [
        {
          id: "fr_live_start_popup_visibility",
          allEnvironments: true,
          type: "experiment",
          hashAttribute: "id",
          coverage: 1,
          trackingKey: featureKey,
          enabled: true,
          description: "A/B test for live start popup visibility",
          values: [
            {value: "false", weight: 0.5, name: "control"},
            {value: "true", weight: 0.5, name: "treatment"}
          ]
        }
      ]
    }
  },
  {upsert: true}
);

const featureCacheQuery = {};
featureCacheQuery["payload.features." + featureKey] = {\$exists: true};
db.sdkconnectioncaches.deleteMany({
  \$or: [
    {key: clientKey},
    {key: sdkConnectionKey},
    featureCacheQuery
  ]
});
JS
}

verify_public_payload() {
  curl -fsS "$GROWTHBOOK_API_URL/api/features/$GROWTHBOOK_CLIENT_KEY" |
    jq -e --arg feature "$FEATURE_KEY" '
      .features[$feature].defaultValue == false and
      .features[$feature].rules[0].variations == [false, true] and
      .features[$feature].rules[0].weights == [0.5, 0.5] and
      .features[$feature].rules[0].meta[0].name == "control" and
      .features[$feature].rules[0].meta[1].name == "treatment"
    ' >/dev/null
}

main() {
  require_cmd docker
  require_cmd curl
  require_cmd jq

  require_growthbook_ready
  ensure_growthbook_first_user
  ensure_growthbook_records
  verify_public_payload

  echo "GrowthBook 初始化完成: $FEATURE_KEY is available via $GROWTHBOOK_CLIENT_KEY"
}

main "$@"
