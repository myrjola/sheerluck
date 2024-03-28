INSERT INTO "users" ("id", "display_name")
VALUES (X'01', 'Test user 1');
INSERT INTO "users" ("id", "display_name")
VALUES (X'02', 'Test user 2');

INSERT INTO "completions" ("id", "user_id", "investigation_target_id", "order", "question", "answer")
VALUES (1, X'01', 'le-bon', 0, 'What is your name?', 'Adolphe Le Bon'),
       (2, X'01', 'le-bon', 1, 'What is your occupation?', 'Bank clerc'),
       (3, X'01', 'le-bon', 2, 'What is your address?', 'Rue Morgue'),
       (4, X'02', 'rue-morgue', 0, 'Where am I?', 'Rue Morgue Murder Scene');
