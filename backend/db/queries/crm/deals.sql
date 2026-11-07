-- name: ListDeals :many
select d.id, d.client_id, c.name as client_name, d.title, d.amount, d.probability, d.expected_close_date,
       d.owner_user_id, u.full_name as owner_name, d.stage, d.stage_changed_at, d.created_at, d.updated_at
from deals d
join clients c on c.id = d.client_id
join users u on u.id = d.owner_user_id
where (sqlc.narg('q')::text is null or d.title ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('client_id')::bigint is null or d.client_id = sqlc.narg('client_id'))
  and (sqlc.narg('stage')::text is null or d.stage = sqlc.narg('stage'))
  and (sqlc.narg('owner_user_id')::bigint is null or d.owner_user_id = sqlc.narg('owner_user_id'))
order by d.created_at desc, d.id desc
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountDeals :one
select count(*)
from deals d
where (sqlc.narg('q')::text is null or d.title ilike '%' || sqlc.narg('q') || '%')
  and (sqlc.narg('client_id')::bigint is null or d.client_id = sqlc.narg('client_id'))
  and (sqlc.narg('stage')::text is null or d.stage = sqlc.narg('stage'))
  and (sqlc.narg('owner_user_id')::bigint is null or d.owner_user_id = sqlc.narg('owner_user_id'));

-- name: GetDeal :one
select d.id, d.client_id, c.name as client_name, d.title, d.amount, d.probability, d.expected_close_date,
       d.owner_user_id, u.full_name as owner_name, d.stage, d.stage_changed_at, d.created_at, d.updated_at
from deals d
join clients c on c.id = d.client_id
join users u on u.id = d.owner_user_id
where d.id = $1;

-- name: GetDealForUpdate :one
select id, stage, title from deals where id = $1 for update;

-- name: CreateDeal :one
insert into deals (client_id, title, amount, probability, expected_close_date, owner_user_id)
values ($1, $2, $3, $4, $5, $6)
returning id;

-- name: UpdateDeal :one
update deals
set title = $2, amount = $3, probability = $4, expected_close_date = $5, owner_user_id = $6, updated_at = now()
where id = $1
returning id;

-- name: SetDealStage :exec
update deals set stage = $2, stage_changed_at = now(), updated_at = now() where id = $1;

-- name: InsertDealStageHistory :exec
insert into deal_stage_history (deal_id, from_stage, to_stage, changed_by, comment)
values ($1, $2, $3, $4, $5);

-- name: ListDealStageHistory :many
select h.id, h.deal_id, h.from_stage, h.to_stage, h.changed_by, u.full_name as changed_by_name, h.changed_at, h.comment
from deal_stage_history h
left join users u on u.id = h.changed_by
where h.deal_id = $1
order by h.changed_at, h.id;

-- name: DealClient :one
select client_id from deals where id = $1;
