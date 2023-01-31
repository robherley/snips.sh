# snips.sh ✂️

[snips.sh](https://snips.sh) is a simple, free, and open source pastebin service.

⚠️ it's still _very_ alpha 🐛 lots of bugs, breaking changes and bad decisions

## 🔑 No passwords, ever.

snips.sh uses SSH public key authentication, so as long as you have keypair, you have an account:

```
$ cat my-amazing-code.go | ssh snips.sh
✅ File Uploaded Successfully!
💳 ID: 7KLqCzRGr
🏋️ Size: 876 B
📁 Type: go
🔗 URL: https://snips.sh/f/7KLqCzRGr
📠 SSH Command: ssh f:7KLqCzRGr@snips.sh
```

now wherever you need the file, just ssh and pipe it to your favorite `$PAGER` or, check out [web ui](https://snips.sh)

```
$ ssh f:7KLqCzRGr@snips.sh | bat
```

snips.sh will try it's best to detect the file type on upload. if not, you can always give it a hint:

```
$ cat README.md | ssh snips.sh -- -ext md
```

## 💣 Time-bombed links

have something super secret to share? you can make it private:

```
$ cat SUPER_SECRET.txt | ssh snips.sh -- -private
```

then mint a signed url with a ttl:

```
$ ssh f:rEyxCKRJi1@snips.sh -- sign -ttl 5m
⏰ Signed file expires: 2023-01-30T22:46:53-05:00
🔗 https://snips.sh/f/rEyxCKRJi1?exp=1675136813&sig=RGs4TbQItOcZ5ShwRq14B7mLPExFxWO5sx3NBz6uC34%3D
```

## 🗑️ Deleting files

don't want it anymore? nuke it:

```
$ ssh f:rEyxCKRJi1@snips.sh -- rm
```

## ✨ Coming soon: Interactive TUI

```
$ ssh snips.sh
```
