DELETE FROM transactions;
DELETE FROM item_images;
DELETE FROM items;
DELETE FROM users WHERE role != 'sysadmin';
DELETE FROM departments;
