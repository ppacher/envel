local describe, it, assert = describe, it, assert
local stream = require("envel.stream")
local Subscriber,
      Subscription,
      Observable,
      Subject,
      AsyncSubject,
      ConnectableObservable =
        stream.Subscriber,
        stream.Subscription,
        stream.Observable,
        stream.Subject,
        stream.AsyncSubject,
        stream.ConnectableObservable

local noop = function() end

-- add all available operators
require("envel.stream.operators")

describe("envel.stream", function()
    describe("subscriptions", function()
        it("should run teardown only once", function()
            local cleanup = 0
            local sub = Subscription.create(function()
                cleanup = cleanup + 1
            end)

            sub:unsubscribe()
            sub:unsubscribe()
            assert.are.equals(cleanup, 1)
        end)

        it("should allow nesting subscriptions", function()
            local s1_called, s2_called, s3_called, s4_called = false, false, false, false

            local s1 = Subscription.create(function() s1_called = true end)
            local s2 = Subscription.create(function() s2_called = true end)
            local s3 = Subscription.create(function() s3_called = true end)
            local s4 = Subscription.create(function() s4_called = true end)

            s1:add(s2)
            s2:add(s3)
            s2:add(s4)

            s4:unsubscribe()
            assert.True(s4_called)
            s4_called = false

            s1:unsubscribe()
            assert.True(s1_called)
            assert.True(s2_called)
            assert.True(s3_called)
            assert.False(s4_called)
        end)

        it('should allow adding custom teardown functions', function()
            local called = false
            local s1 = Subscription.create(function() end)
            s1:add(function()
                called = true
            end)

            s1:unsubscribe()
            assert.True(called)
        end)

        it('should allow adding custom subscription objects', function()
            local custom = {}
            local called = false
            function custom.unsubscribe() called = true end

            local s1 = Subscription.create()
            s1:add(custom)
            s1:unsubscribe()
            assert.True(called)
        end)
    end)

    describe("subscriber", function()

        it("should extend subscription", function()
            assert.are.same(getmetatable(Subscriber).__index, Subscription)
        end)

        -- check at least if the nested subscription handling works. In combination with
        -- the above check we can be mostly sure that it correctly extends Subscription
        it("should allow nested subscriptions", function()
            local sub = Subscriber.create(noop)

            local called = false
            local s1 = Subscription.create(function() called = true end)
            sub:add(s1)
            sub:unsubscribe()
            assert.True(called)
        end)

        it('should allow emitting values', function()
            local values = {}
            local sub = Subscriber.create(function(_, v) table.insert(values, v) end)

            sub:next(1)
            sub:next(2)
            sub:next(1)

            assert.are.same({1, 2, 1}, values)
        end)

        it('should emit errors and complete', function()
            local emitted = 0
            local error = nil

            local sub = Subscriber.create(function() emitted = emitted + 1 end, function(_, e) error = e end)
            sub:next(1)
            sub:error("test")
            sub:next(1)

            assert.are.equals(emitted, 1)
            assert.are.equals(error, "test")
            assert.True(sub.closed) -- this comes from subscription
            assert.True(sub._is_stopped) -- this comes from subscriber
        end)
    end)

    describe("observable", function()
        it("should allow subscriptions", function()
            local subscribed = false
            local unsubscribed = false

            local obs = Observable.create(function(observer)
                subscribed = true

                observer:next(1)
                observer:next(2)

                return function() unsubscribed = true end
            end)

            local emitted = 0
            local sub = obs:subscribe(function() emitted = emitted + 1 end)

            assert.True(subscribed)
            assert.False(unsubscribed)
            assert.are.equals(emitted, 2)

            sub:unsubscribe()
            assert.True(unsubscribed)
        end)

        it("should unsubscribe on completion", function()
            local subscribed = false
            local unsubscribed = false

            local obs = Observable.create(function(observer)
                subscribed = true

                observer:next(1)
                observer:complete()

                return function() unsubscribed = true end
            end)

            local emitted = 0
            obs:subscribe(function() emitted = emitted + 1 end)

            assert.True(subscribed)
            assert.True(unsubscribed)
            assert.are.equals(emitted, 1)
        end)

        it("should unsubscribe on error", function()
            local subscribed = false
            local unsubscribed = false

            local obs = Observable.create(function(observer)
                subscribed = true

                observer:next(1)
                observer:error("test")

                return function() unsubscribed = true end
            end)

            local emitted = 0
            obs:subscribe(function() emitted = emitted + 1 end, noop)

            assert.True(subscribed)
            assert.True(unsubscribed)
            assert.are.equals(emitted, 1)
        end)

        it("should be chainable", function()
            local obs = Observable.create(function(observer)
                observer:next(1)
                return noop
            end)

            local obs2 = obs:lift(function(sink, source)
                return source:subscribe(sink)
            end)

            local value
            obs2:subscribe(function(_, val) value = val end)

            assert.are.equals(value, 1)
        end)

        it('should not emit values when unsubscribed', function()
            local _observer
            local emitted = 0
            local obs = Observable.create(function(observer)
                _observer = observer
            end)

            local sub = obs:subscribe(function() emitted = emitted + 1 end)


            _observer:next(1)
            _observer:next(1)
            sub:unsubscribe()

            _observer:next(1)
            assert.are.equals(emitted, 2)
        end)
    end)

    describe("subject", function()
        it("should emit values to all subscribers", function()
            local subject = Subject.create()

            local s1_called = 0
            subject:subscribe(function() s1_called = s1_called + 1 end)
            local s2_called = 0
            subject:subscribe(function() s2_called = s2_called + 1 end)

            subject:next(1)
            subject:next(1)

            assert.are.equals(s1_called, 2)
            assert.are.equals(s2_called, 2)
        end)

        it('should no emit to unsubscribed subscriptions', function()
            local subject = Subject.create()

            local s1_called = 0
            subject:subscribe(function() s1_called = s1_called + 1 end)
            local s2_called = 0
            local s2 = subject:subscribe(function() s2_called = s2_called + 1 end)

            subject:next(1)
            s2:unsubscribe()
            subject:next(1)

            assert.are.equals(s1_called, 2)
            assert.are.equals(s2_called, 1)
        end)
    end)

    describe("AsyncSubject", function()
        it("should only emit when completed", function()
            local called = 0
            local handler = function() called = called + 1 end

            local subj = AsyncSubject.create()

            subj:subscribe(handler)

            subj:next(1)
            subj:next(1)
            subj:next(1)
            subj:complete()

            assert.are.equals(1, called)

            -- should emit immediately as the async subject has completed
            subj:subscribe(handler)
            assert.are.equals(2, called)
        end)
    end)

    describe("ConnectableObservable", function()
        it("should subscribe upon connect and complete with source", function()
            local source_subscribed = false
            local source_unsubscribed = false
            local source_observer = nil
            local source = Observable.create(function(observer)
                source_subscribed = true
                source_observer = observer
                return function()
                    source_unsubscribed = true
                end
            end)

            local s1_emitted = 0
            local s2_emitted = 0

            local connectable = ConnectableObservable.create(source)
            local sub1 = connectable:subscribe(function()
                s1_emitted = s1_emitted + 1
            end)

            assert.False(source_subscribed)
            local sub2 = connectable:subscribe(function()
                s2_emitted = s2_emitted + 1
            end)

            assert.False(source_subscribed)

            local connection = connectable:connect()
            assert.True(source_subscribed)
            assert.False(source_unsubscribed)

            source_observer:next(1)
            source_observer:next(1)

            assert.are.equals(2, s1_emitted)
            assert.are.equals(2, s2_emitted)

            sub2:unsubscribe()

            source_observer:next(1)
            assert.are.equals(3, s1_emitted)
            assert.are.equals(2, s2_emitted)


            connection:unsubscribe()
            assert.True(source_unsubscribed)
            source_subscribed = false
            source_unsubscribed = false
            source_observer = nil

            connectable:connect()
            assert.True(source_subscribed)
            assert.False(source_unsubscribed)

            source_observer:next(1)
            assert.are.equals(4, s1_emitted)
            assert.are.equals(2, s2_emitted)

            source_observer:complete()
            assert.True(source_unsubscribed)

            sub1:unsubscribe()
        end)
    end)

    describe("operators", function()
        local observer
        local obs = Observable.create(function(_observer) observer = _observer end)

        describe("map", function()
            it("should allow trasforming values", function()
                local values = {}
                obs:map(function(value)
                    return value + 1
                end):subscribe(function(_, value)
                    table.insert(values, value)
                end)

                observer:next(3)
                observer:next(10)

                assert.are.same({4, 11}, values)

            end)
        end)

        describe("multicast", function()
            it("should publish values to all subscribers", function()
                local trigger = nil
                local obs = Observable.create(function(observer)
                    trigger = function()
                        observer:next(2)
                    end

                    observer:next(1)
                end):multicast()

                assert.are.equals("ConnectableObservable", tostring(obs))

                local s1_called = 0
                local s1 = obs:subscribe(function() s1_called = s1_called + 1 end)

                obs:connect()

                assert.are.equals(s1_called, 1)

                local s2_called = 0
                local s2 = obs:subscribe(function() s2_called = s2_called + 1 end)

                trigger()
                trigger()

                assert.are.equals(s1_called, 3)
                assert.are.equals(s2_called, 2)
            end)
        end)

        describe("share", function()
            it("should share values to all subscribers, makeing it hot", function()
                local trigger = nil
                local obs = Observable.create(function(observer)
                    trigger = function()
                        observer:next(2)
                    end

                    observer:next(1)
                end):share()

                local s1_called = 0
                local s1 = obs:subscribe(function() s1_called = s1_called + 1 end)

                assert.are.equals(s1_called, 1)

                local s2_called = 0
                local s2 = obs:subscribe(function() s2_called = s2_called + 1 end)

                trigger()
                trigger()

                assert.are.equals(s1_called, 3)
                assert.are.equals(s2_called, 2)
            end)

            it("should unsubscribe from source when refcount drops to zero", function()
                local subscribed, unsubscribed = false, false
                local obs = Observable.create(function(observer)
                    subscribed = true
                    return function() unsubscribed = true end
                end):share()

                assert.False(subscribed)
                assert.False(unsubscribed)

                local s1 = obs:subscribe()
                assert.True(subscribed)
                assert.False(unsubscribed)

                local s2 = obs:subscribe()
                assert.False(unsubscribed)

                s1:unsubscribe()
                assert.False(unsubscribed)
                s2:unsubscribe()
                assert.True(unsubscribed)
            end)
        end)
    end)
end)
