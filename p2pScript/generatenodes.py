#coding=gbk
from cgi import test
import platform
from typing import List 
import utils.util as ut
import content.content as ct
import os

## CONSTS
OS_SEPARATOR = os.sep   #根据系统选择分隔符#
CURRENT_FILE_PATH = os.path.abspath(__file__)
CURRENT_DIR = os.path.dirname(CURRENT_FILE_PATH)
WORK_PATH = CURRENT_DIR
TEMPLATE_PATH = WORK_PATH + OS_SEPARATOR + "templatenode.zip"

CLIENT_PROGRAM = ""
SYSTEM_PLATFORM = platform.system()
if SYSTEM_PLATFORM == 'Linux':
    CLIENT_PROGRAM = "whisper_client"
elif SYSTEM_PLATFORM == 'Windows':
    CLIENT_PROGRAM = "whisper_client.exe"


class node(object):
    def __init__(self,name:str,dirpath:str,listenaddr:str = '127.0.0.1:30301'):  #指定节点监听的地址#
        self.name = name
        self.nodedir = dirpath + OS_SEPARATOR + name
        self.pubkey = self.nodedir + OS_SEPARATOR + "pubkeys" + OS_SEPARATOR +  "self.pubkey"   #生成公钥#
        self.bootpath = self.nodedir + OS_SEPARATOR + "bootnodes" + OS_SEPARATOR + "nodes.txt"	#生成节点列表文件
        self.actionlist = self.nodedir + OS_SEPARATOR + "action" + OS_SEPARATOR + "actionlist.txt"
        self.url = ""
        self.listenaddr = listenaddr


    def unzip(self):
        ut.unzip_file(TEMPLATE_PATH,self.nodedir)


    def readurl(self):
        temp_path = self.nodedir +OS_SEPARATOR+"bootnodes"+ OS_SEPARATOR + "self.txt"
        try:
            f = open(temp_path)
            self.url = f.readline()
            f.close()
        except:
            print("Can't read url")


    def keygenerate(self):
        cmd = self.nodedir + OS_SEPARATOR + CLIENT_PROGRAM + " keygenerate"
        os.system(cmd)


    def saveurl(self):
        cmd = self.nodedir + OS_SEPARATOR + CLIENT_PROGRAM + " urlsave -addr " + self.listenaddr
        os.system(cmd)

    def addbootnode(self,nodeurl:str):
        try:
            f = open(self.bootpath,"a+")
            f.write(nodeurl+'\n')
            f.close()
        except:
            print("Can't add boot node")
    
    def addpubkey(self,srcpath:str,nodename:str):
        destpath = self.nodedir + OS_SEPARATOR + "pubkeys" + OS_SEPARATOR + nodename + ".pubkey"
        ut.copy_file(srcpath,destpath)

    def writeactions(self,actions:str):
        try:
            f = open(self.actionlist,"a+")
            f.write(actions)
            f.close
        except:
            print("Can't write actions")
    #生成节点运行脚本
    def writenodetestscript(self,filename:str):   #linux没有pause命令
        win_cont = "." + OS_SEPARATOR + CLIENT_PROGRAM + " clientstart -name " + self.name + \
            " -discovery -addr "+self.listenaddr+" -automode" + "\n" + "pause"
        lin_cont = "#!/bin/bash"+"\n"+"."+ OS_SEPARATOR+ CLIENT_PROGRAM + " clientstart -name " + self.name + \
            " -discovery -addr "+self.listenaddr+" -automode"

        writescript(self.nodedir,filename,[win_cont,lin_cont])

    def nodeinit(self):
        self.unzip()
        self.keygenerate()
        self.saveurl()
        self.readurl()


    

        

    
## cont[0]: windows case, cont[1]:linux case
def writescript(dirpath:str,filename:str,cont:List[str]):
    script_path = ""
    if SYSTEM_PLATFORM == 'Windows':
        script_path = dirpath + OS_SEPARATOR + filename + ".bat"
        try:
            f = open(script_path,"a+")
            f.write(cont[0])
            f.close
        except:
            print("Can't write test script")

    elif SYSTEM_PLATFORM == 'Linux':
        script_path = dirpath + OS_SEPARATOR + filename + ".sh"
        try:
            f = open(script_path,"a+")
            f.write(cont[1])
            f.close
        except:
            print("Can't write test script")
    
    




     
#只有两节点的测试#
def test1(force: bool = False):
    testdir = WORK_PATH + OS_SEPARATOR + "test1"

    if force:
        ut.remove_dir(testdir)
    ut.create_dir(testdir)

    node1 = node("node1",testdir,"127.0.0.1:30301")  #指定node1监听端口#
    # node1.unzip()

    node2 = node("node2",testdir,"127.0.0.1:30302")	 #指定node2监听端口#
    # node2.unzip()

    # node1.keygenerate()
    # node1.saveurl()
    # node2.keygenerate()
    # node2.saveurl()

    # node1.readurl()
    # node2.readurl()
    node1.nodeinit()
    node2.nodeinit()

    node2.addbootnode(node1.url)

    node1.addpubkey(node2.pubkey,node2.name)
    node2.addpubkey(node1.pubkey,node1.name)

    node1.writeactions(ct.t1_n1)
    node2.writeactions(ct.t1_n2)

    node1.writenodetestscript('test')   #指定node1脚本名称#
    node2.writenodetestscript('test')	#指定node2脚本名称#

    writescript(testdir,'teststart',[ct.t1_tot_test_win,ct.t1_tot_test_lin]) #根据系统确定脚本后缀，bat or sh#





#有三节点的测试#
def test2(force: bool = False):
    testdir = WORK_PATH + OS_SEPARATOR + "test2"

    if force:
        ut.remove_dir(testdir)
    ut.create_dir(testdir)

    node1 = node("node1",testdir,"127.0.0.1:30301")
    node2 = node("node2",testdir,"127.0.0.1:30302")
    node3 = node("node3",testdir,"127.0.0.1:30303")

    node1.nodeinit()
    node2.nodeinit()
    node3.nodeinit()

    node2.addbootnode(node1.url)
    node3.addbootnode(node2.url)

    node1.addpubkey(node2.pubkey,node2.name)
    node1.addpubkey(node3.pubkey,node3.name)
    node2.addpubkey(node1.pubkey,node1.name)
    node2.addpubkey(node3.pubkey,node3.name)
    node3.addpubkey(node1.pubkey,node1.name)
    node3.addpubkey(node2.pubkey,node2.name)

    node1.writeactions(ct.t2_n1)
    node2.writeactions(ct.t2_n2)
    node3.writeactions(ct.t2_n3)

    node1.writenodetestscript('test')
    node2.writenodetestscript('test')
    node3.writenodetestscript('test')

    writescript(testdir,'teststart',[ct.t2_tot_test_win,ct.t2_tot_test_lin])





if __name__ == "__main__":
    test1(force=True)
    test2(force=True)

