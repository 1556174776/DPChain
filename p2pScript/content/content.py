################################################## test1
t1_n1 = '''SELFKEYREGISTER
REGISTERPUBKEY
WATCH|SELF|D
SLEEP|10
SEND|SELF|node2|D|Hello, I am Node1.
SLEEP|10
'''

t1_n2 = '''SELFKEYREGISTER
REGISTERPUBKEY
WATCH|SELF|D
SLEEP|3
SEND|SELF|node1|D|Hello, I am Node2.
SLEEP|20
'''

t1_tot_test_win = '''@echo off 
start /D ".\\node1" test.bat
start /D ".\\node2" test.bat
'''

t1_tot_test_lin = '''
sh ./node1/test.sh
sh ./node2/test.sh
'''


################################################## test2
t2_n1 = '''SELFKEYREGISTER
REGISTERPUBKEY
WATCH|SELF|D
SLEEP|10
SEND|SELF|node2|D|Hello, I am Node1.
SLEEP|10
'''

t2_n2 = '''SELFKEYREGISTER
REGISTERPUBKEY
WATCH|SELF|D
SLEEP|25
'''

t2_n3 = '''SELFKEYREGISTER
REGISTERPUBKEY
SLEEP|3
SEND|SELF|node1|D|Hello, I am Node3.
SLEEP|20
'''

t2_tot_test_win = '''@echo off 
start /D ".\\node1" test.bat
start /D ".\\node2" test.bat
start /D ".\\node3" test.bat
'''

t2_tot_test_lin = '''
sh ./node1/test.sh
sh ./node2/test.sh
sh ./node3/test.sh
'''



