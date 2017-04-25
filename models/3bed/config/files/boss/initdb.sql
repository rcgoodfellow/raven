

replace into nodes 
  set node_id='stem',
      phys_nodeid='stem',
      type='cx1',
      role='ctrlswitch'
;

replace into nodes 
  set node_id='leaf',
      phys_nodeid='leaf',
      type='cx1',
      role='testswitch'
;

replace into switch_stack_types (
    stack_id,
    stack_type,
    snmp_community,
    min_vlan,
    max_vlan,
    leader
  ) values ('Control', 'generic', 'private', 2, 997, 'stem'),
           ('Experiment', 'generic', 'private', 2, 997, 'leaf')
;

replace into switch_stacks (
  node_id,
  stack_id,
  is_primary
  ) values ('leaf', 'Experiment', 1),
           ('stem', 'Control', 0)
;
