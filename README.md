# rbi
mac cgo
```bash
 CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CGO_LDFLAGS="-static" go build -v -ldflags="-w -s" -trimpath  -o rbi
```

### 前端配置
#### 安装依赖
```bash
cd web
yarn
```
#### 环境变量
开发环境适用
``.env.development``
生产环境适用`.env.production`
项目配置`src/settings/projectSetting.ts`
组建配置`src/settings/componentSetting.ts`
主题配置`src/settings/designSetting.ts`
路由配置 在 src/router/modules 内的 .ts 文件会被视为一个路由模块。
配置首页 `src/enums/pageEnum.ts`