import os
import shutil
import zipfile


OS_SEPARATOR = os.sep
## zip文件
def zip_file(src_dir: str,des_dir:str):
    zip_name = des_dir +'.zip'
    z = zipfile.ZipFile(zip_name,'w',zipfile.ZIP_DEFLATED)
    for dirpath, dirnames, filenames in os.walk(src_dir):
        fpath = dirpath.replace(src_dir,'')
        fpath = fpath and fpath + os.sep or ''
        for filename in filenames:
            z.write(os.path.join(dirpath, filename),fpath+filename)
            print ('==压缩成功==')
    z.close()

## zip解压文件
def unzip_file(zip_src, dst_dir):
    r = zipfile.is_zipfile(zip_src)
    if r:     
        fz = zipfile.ZipFile(zip_src, 'r')
        for file in fz.namelist():
            fz.extract(file, dst_dir)
            os.chmod(dst_dir + OS_SEPARATOR +file.rstrip(OS_SEPARATOR),0o755)  #解压成功后，将所有文件权限修改为755 rwxr-xr-x
            #print(dst_dir + OS_SEPARATOR +file.rstrip(OS_SEPARATOR))
                   
    else:
        print('This is not zip')


## 创建文件夹，要求使用绝对路径
def create_dir(dirname:str):
    isExists = os.path.exists(dirname)
    if not isExists:
        oldmask = os.umask(000)
        os.makedirs(dirname,mode=0o777)
        os.umask(oldmask)
        print(dirname+"创建成功")
        return 
    else:
        print(dirname+"已存在")
        return 

## 删除文件夹，要求使用绝对路径
def remove_dir(dirname:str):
    isExists = os.path.exists(dirname)
    if isExists:
        shutil.rmtree(dirname)


## 复制文件，要求使用绝对路径
def copy_file(srcpath:str,destpath:str):
    if not os.path.exists(srcpath):
        print("copy fail: no such src file")
        return
    if os.path.exists(destpath):
        print("copy fail: dest file has existed")
        return

    shutil.copyfile(srcpath, destpath)

