# spotify-local-manager
spotify本地文件管理工具.可以对本地音频文件进行分类,分离未处理音频,同步到spotify的歌单等.


## FEATURES

> [!important]
>
> 1. 将spotify本地文件夹的已分类歌曲筛选出来 放在另外一个文件夹(同级) spotify_local_temp
> 2. 打开spotify
> 3. 将这些未分类的音乐分类
> 4. 监听cmd窗口关闭事件 一旦关闭 就将spotify_local_temp的已分类好的音乐文件移回
> 5. 分类完成

## INSTALL && CONFIG

- 下载最新的[Release](https://github.com/nichuanfang/spotify-local-manager/releases)
- 解压至`spotify本地文件夹`同目录下,保证`spotify-local-manager.exe`同级目录有`spotify_local`文件夹,这是存储你本地音频的文件夹
- 使用[music-tool-kit](https://pypi.org/project/music-tool-kit/)下载mp3文件

> [!NOTE]
>
> * `spotify_local`可以与[阿里云盘桌面端](https://www.alipan.com/)的文件夹同步配合食用~
> * `spotify_local_temp`是存储待分类和分类错误的音频文件的,参考`http://127.0.0.1:9999`的分类预览页面,可以打开该文件夹进行分类
> * 当不需要分类时应当及时关闭cmd窗口防止受到SpotifyApi的[rate limit](https://developer.spotify.com/documentation/web-api/concepts/rate-limits)
