
replace into node_types set 
  class='pc',
  type='qnode'
;

replace into node_type_attributes values
  ('qnode', 'adminmfs_osid',
    (select osid from os_info where osname='linux-mfs'), 'integer'),
  ('qnode', 'bios_waittime', 10, 'integer'),
  ('qnode', 'bootdisk_unit', 0, 'integer'),
  ('qnode', 'cluster', 'vbed3', 'string'),
  ('qnode', 'console', 'com2', 'string'),
  ('qnode', 'control_interface', 'eth0', 'string'),
  ('qnode', 'control_network', 0, 'integer'),
  ('qnode', 'default_imageid',
    (select osid from os_info where osname='Ubuntu1604-STD'), 'integer'),
  ('qnode', 'delay_capacity', 0, 'iteger'),
  ('qnode', 'diskloadmfs_osid',
    (select osid from os_info where osname='linux-mfs'), 'integer'),
  ('qnode', 'disksize', 100, 'float'),
  ('qnode', 'disktype', 'ad', 'string'),
  ('qnode', 'frequency', '1000', 'integer'),
  ('qnode', 'imageable', 1, 'boolean'),
  ('qnode', 'max_interfaces', 2, 'integer'),
  ('qnode', 'memory', 40960, 'integer'),
  ('qnode', 'power_delay', 10, 'integer'),
  ('qnode', 'processor', 'kvm-vproc', 'string'),
  ('qnode', 'rebootable', 1, 'boolean'),
  ('qnode', 'simnode_capacity', 0, 'integer'),
  ('qnode', 'special_hw', 0, 'integer'),
  ('qnode', 'trivlink_maxspeed', 1000, 'integer'),
  ('qnode', 'virtnode_capacity', 5, 'integer')
;

replace into osidtoimageid 
  select default_osid as osid, 'qnode' as type, imageid from images 
  where imagename='Ubuntu1604-STD'
;
