-- name: GetInvestigationTarget :one
SELECT *
FROM investigation_targets
WHERE id = :investigationTargetID;

-- name: ListInvestigationTargets :many
SELECT *
FROM investigation_targets
WHERE case_id = :caseID AND type = :type;
