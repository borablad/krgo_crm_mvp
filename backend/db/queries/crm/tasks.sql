-- name: ListTasks :many
select t.id, t.title, t.description, t.assignee_user_id, u.full_name as assignee_name, t.due_date, t.status,
       t.client_id, c.name as client_name, t.deal_id, d.title as deal_title, t.created_by, t.done_at, t.created_at, t.updated_at
from tasks t
join users u on u.id = t.assignee_user_id
left join clients c on c.id = t.client_id
left join deals d on d.id = t.deal_id
where (sqlc.narg('assignee_user_id')::bigint is null or t.assignee_user_id = sqlc.narg('assignee_user_id'))
  and (sqlc.narg('status')::text is null or t.status = sqlc.narg('status'))
  and (sqlc.narg('client_id')::bigint is null or t.client_id = sqlc.narg('client_id'))
  and (sqlc.narg('deal_id')::bigint is null or t.deal_id = sqlc.narg('deal_id'))
  and (not sqlc.arg('overdue')::boolean or (t.status = 'open' and t.due_date < sqlc.arg('today')::date))
order by t.due_date, t.id
limit sqlc.arg('lim') offset sqlc.arg('off');

-- name: CountTasks :one
select count(*)
from tasks t
where (sqlc.narg('assignee_user_id')::bigint is null or t.assignee_user_id = sqlc.narg('assignee_user_id'))
  and (sqlc.narg('status')::text is null or t.status = sqlc.narg('status'))
  and (sqlc.narg('client_id')::bigint is null or t.client_id = sqlc.narg('client_id'))
  and (sqlc.narg('deal_id')::bigint is null or t.deal_id = sqlc.narg('deal_id'))
  and (not sqlc.arg('overdue')::boolean or (t.status = 'open' and t.due_date < sqlc.arg('today')::date));

-- name: GetTask :one
select t.id, t.title, t.description, t.assignee_user_id, u.full_name as assignee_name, t.due_date, t.status,
       t.client_id, c.name as client_name, t.deal_id, d.title as deal_title, t.created_by, t.done_at, t.created_at, t.updated_at
from tasks t
join users u on u.id = t.assignee_user_id
left join clients c on c.id = t.client_id
left join deals d on d.id = t.deal_id
where t.id = $1;

-- name: GetTaskForUpdate :one
select id, status, title from tasks where id = $1 for update;

-- name: CreateTask :one
insert into tasks (title, description, assignee_user_id, due_date, client_id, deal_id, created_by)
values ($1, $2, $3, $4, $5, $6, $7)
returning id;

-- name: UpdateTask :one
update tasks
set title = $2, description = $3, assignee_user_id = $4, due_date = $5, client_id = $6, deal_id = $7, updated_at = now()
where id = $1
returning id;

-- name: CompleteTask :exec
update tasks set status = 'done', done_at = now(), updated_at = now() where id = $1;

-- name: CancelTask :exec
update tasks set status = 'cancelled', updated_at = now() where id = $1;
