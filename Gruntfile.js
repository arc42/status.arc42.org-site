module.exports = function (grunt) {

    grunt.initConfig({
        pkg: grunt.file.readJSON('package.json'),

        uglify: {

            options: {
                banner: '// <%= pkg.version, pkg.name, pkg.url %> \n' +
                    '// <%= pkg.version %> (<%= grunt.template.today("yyyy-mm-dd, HH:MM") %>)',
                compress: {
                    drop_console: true // strip out all console.log statements
                },
                output: {
                    comments: false // strip out all comments
                }
            },
            dist: {
                src: ["src/js/createMap.js",],
                dest: "dist/createMap.min.js"
            },
            a: {
                src: "src/js/a.js",
                dest: "dist/a.min.js"
            }

        }

    });


    //grunt.loadNpmTasks('grunt-contrib-obfuscator');
    grunt.loadNpmTasks('grunt-contrib-uglify');
    grunt.registerTask('default', ['uglify']);


}
;