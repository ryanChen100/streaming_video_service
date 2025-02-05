const MONGO_USER = process.env.MONGO_INITDB_ROOT_USERNAME;
const MONGO_PASSWORD = process.env.MONGO_INITDB_ROOT_PASSWORD;

db = db.getSiblingDB("member_db");
db.createUser({
  user: MONGO_USER,
  pwd: MONGO_PASSWORD,
  roles: [{ role: "readWrite", db: "member_db" }]
});

db = db.getSiblingDB("chat_db");
db.createUser({
  user: MONGO_USER,
  pwd: MONGO_PASSWORD,
  roles: [{ role: "readWrite", db: "chat_db" }]
});

db = db.getSiblingDB("streaming_db");
db.createUser({
  user: MONGO_USER,
  pwd: MONGO_PASSWORD,
  roles: [{ role: "readWrite", db: "streaming_db" }]
});